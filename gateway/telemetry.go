package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// A2A Gateway Metrics
type A2AMetrics struct {
	// Request metrics
	RequestsTotal     otelmetric.Int64Counter
	RequestDuration   otelmetric.Float64Histogram
	RequestsInFlight  otelmetric.Int64UpDownCounter
	
	// Task metrics
	TasksCreated      otelmetric.Int64Counter
	TasksCompleted    otelmetric.Int64Counter
	TasksFailed       otelmetric.Int64Counter
	TaskDuration      otelmetric.Float64Histogram
	ActiveTasks       otelmetric.Int64UpDownCounter
	
	// Agent metrics
	AgentRequests     otelmetric.Int64Counter
	AgentErrors       otelmetric.Int64Counter
	AgentResponseTime otelmetric.Float64Histogram
	
	// Error metrics
	ErrorsTotal       otelmetric.Int64Counter
	ErrorsByCategory  otelmetric.Int64Counter
	
	// System metrics
	RedisConnections  otelmetric.Int64UpDownCounter
	TemporalHealth    otelmetric.Int64UpDownCounter
}

var (
	tracer  oteltrace.Tracer
	metrics *A2AMetrics
)

// Initialize OpenTelemetry
func initTelemetry() (*sdkmetric.MeterProvider, func(), error) {
	serviceName := "a2a-gateway"
	serviceVersion := "0.5.0"
	
	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.ServiceNamespace("a2a"),
			attribute.String("environment", getEnv("NODE_ENV", "production")),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing
	if err := setupTracing(res); err != nil {
		return nil, nil, fmt.Errorf("failed to setup tracing: %w", err)
	}

	// Setup metrics
	meterProvider, err := setupMetrics(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup metrics: %w", err)
	}

	// Initialize A2A metrics
	if err := initA2AMetrics(); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize A2A metrics: %w", err)
	}

	// Setup global propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Printf("✅ OpenTelemetry initialized for %s v%s", serviceName, serviceVersion)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}

	return meterProvider, cleanup, nil
}

func setupTracing(res *resource.Resource) error {
	// Only setup tracing if enabled
	if !isTracingEnabled() {
		log.Printf("🔍 Distributed tracing disabled")
		return nil
	}

	// Jaeger exporter
	jaegerEndpoint := getEnv("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces")
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(getSamplingRate())),
	)

	otel.SetTracerProvider(tp)
	tracer = otel.Tracer("a2a-gateway")

	log.Printf("🔍 Distributed tracing enabled (Jaeger: %s)", jaegerEndpoint)
	return nil
}

func setupMetrics(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	// Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Meter provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	otel.SetMeterProvider(provider)
	log.Printf("📊 Prometheus metrics enabled at /metrics")

	return provider, nil
}

func initA2AMetrics() error {
	meter := otel.Meter("a2a-gateway")

	var err error
	metrics = &A2AMetrics{}

	// Request metrics
	metrics.RequestsTotal, err = meter.Int64Counter(
		"a2a_requests_total",
		otelmetric.WithDescription("Total number of A2A requests"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create requests_total counter: %w", err)
	}

	metrics.RequestDuration, err = meter.Float64Histogram(
		"a2a_request_duration_seconds",
		otelmetric.WithDescription("A2A request duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create request_duration histogram: %w", err)
	}

	metrics.RequestsInFlight, err = meter.Int64UpDownCounter(
		"a2a_requests_in_flight",
		otelmetric.WithDescription("Current number of A2A requests in flight"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create requests_in_flight counter: %w", err)
	}

	// Task metrics
	metrics.TasksCreated, err = meter.Int64Counter(
		"a2a_tasks_created_total",
		otelmetric.WithDescription("Total number of tasks created"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tasks_created counter: %w", err)
	}

	metrics.TasksCompleted, err = meter.Int64Counter(
		"a2a_tasks_completed_total",
		otelmetric.WithDescription("Total number of tasks completed"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tasks_completed counter: %w", err)
	}

	metrics.TasksFailed, err = meter.Int64Counter(
		"a2a_tasks_failed_total",
		otelmetric.WithDescription("Total number of tasks failed"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tasks_failed counter: %w", err)
	}

	metrics.TaskDuration, err = meter.Float64Histogram(
		"a2a_task_duration_seconds",
		otelmetric.WithDescription("Task execution duration in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create task_duration histogram: %w", err)
	}

	metrics.ActiveTasks, err = meter.Int64UpDownCounter(
		"a2a_active_tasks",
		otelmetric.WithDescription("Current number of active tasks"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create active_tasks counter: %w", err)
	}

	// Agent metrics
	metrics.AgentRequests, err = meter.Int64Counter(
		"a2a_agent_requests_total",
		otelmetric.WithDescription("Total number of agent requests"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create agent_requests counter: %w", err)
	}

	metrics.AgentErrors, err = meter.Int64Counter(
		"a2a_agent_errors_total",
		otelmetric.WithDescription("Total number of agent errors"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create agent_errors counter: %w", err)
	}

	metrics.AgentResponseTime, err = meter.Float64Histogram(
		"a2a_agent_response_time_seconds",
		otelmetric.WithDescription("Agent response time in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create agent_response_time histogram: %w", err)
	}

	// Error metrics
	metrics.ErrorsTotal, err = meter.Int64Counter(
		"a2a_errors_total",
		otelmetric.WithDescription("Total number of A2A errors"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create errors_total counter: %w", err)
	}

	metrics.ErrorsByCategory, err = meter.Int64Counter(
		"a2a_errors_by_category_total",
		otelmetric.WithDescription("Total number of A2A errors by category"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create errors_by_category counter: %w", err)
	}

	// System metrics
	metrics.RedisConnections, err = meter.Int64UpDownCounter(
		"a2a_redis_connections",
		otelmetric.WithDescription("Current number of Redis connections"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create redis_connections counter: %w", err)
	}

	metrics.TemporalHealth, err = meter.Int64UpDownCounter(
		"a2a_temporal_health",
		otelmetric.WithDescription("Temporal service health (1=healthy, 0=unhealthy)"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create temporal_health gauge: %w", err)
	}

	log.Printf("📊 A2A metrics initialized")
	return nil
}

// Helper functions
func isTracingEnabled() bool {
	return getEnv("ENABLE_TRACING", "false") == "true"
}

func getSamplingRate() float64 {
	// Default to 10% sampling in production
	if getEnv("NODE_ENV", "production") == "development" {
		return 1.0 // 100% sampling in development
	}
	return 0.1 // 10% sampling in production
}

// Metrics helper functions
func RecordRequest(ctx context.Context, method string, statusCode int, duration time.Duration) {
	if metrics == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.Int("status_code", statusCode),
	}

	metrics.RequestsTotal.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
	metrics.RequestDuration.Record(ctx, duration.Seconds(), otelmetric.WithAttributes(attrs...))
}

func RecordTaskCreated(ctx context.Context, agentID string) {
	if metrics == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("agent_id", agentID),
	}

	metrics.TasksCreated.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
	metrics.ActiveTasks.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
}

func RecordTaskCompleted(ctx context.Context, agentID string, duration time.Duration, success bool) {
	if metrics == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("agent_id", agentID),
		attribute.Bool("success", success),
	}

	if success {
		metrics.TasksCompleted.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
	} else {
		metrics.TasksFailed.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
	}

	metrics.TaskDuration.Record(ctx, duration.Seconds(), otelmetric.WithAttributes(attrs...))
	metrics.ActiveTasks.Add(ctx, -1, otelmetric.WithAttributes(
		attribute.String("agent_id", agentID),
	))
}

func RecordA2AError(ctx context.Context, errorCode int, category string) {
	if metrics == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.Int("error_code", errorCode),
		attribute.String("category", category),
	}

	metrics.ErrorsTotal.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
	metrics.ErrorsByCategory.Add(ctx, 1, otelmetric.WithAttributes(attrs...))
}

// HTTP handler to expose Prometheus metrics
func CreateMetricsHandler() http.Handler {
	return promhttp.Handler()
}