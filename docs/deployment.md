# Deployment Guide

This guide covers deploying the Temporal A2A Gateway in various environments, from development to production.

## Prerequisites

### System Requirements
- Docker 20.10+ and Docker Compose 2.0+
- Go 1.24+ (for building from source)
- Python 3.12+ (for examples and tools)
- 4GB RAM minimum (8GB recommended for production)
- Network access to Temporal and Redis services

### External Dependencies
- **Temporal Server**: Workflow orchestration engine
- **Redis**: Caching and task indexing (optional but recommended)
- **Agent Registry**: Service discovery (optional)

## Development Deployment

### Quick Start with Docker Compose

1. Clone and prepare the repository:
```bash
git clone https://github.com/standel/temporal-a2a-gateway
cd temporal-a2a-gateway
```

2. Start the complete stack:
```bash
docker-compose -f examples/docker-compose.yml up -d
```

3. Verify deployment:
```bash
# Check gateway health
curl http://localhost:8080/health

# Check Temporal UI
open http://localhost:8233

# Run example
cd examples/python
python google_a2a_sdk_integration_example.py
```

### Development Environment Variables
```bash
# .env file for development
TEMPORAL_HOST=localhost
TEMPORAL_PORT=7233
TEMPORAL_NAMESPACE=default
A2A_PORT=8080
REDIS_URL=redis://localhost:6379
LOG_LEVEL=debug
JWT_SECRET=development-secret-not-for-production
```

## Production Deployment

### Architecture Overview
```
[Load Balancer] → [A2A Gateway] → [Temporal] → [Agent Workers]
                       ↓
                   [Redis Cache]
```

### Docker Deployment

#### Production Docker Compose
```yaml
version: '3.8'

services:
  gateway:
    image: temporal-a2a-gateway:latest
    ports:
      - "8080:8080"
    environment:
      - TEMPORAL_HOST=${TEMPORAL_HOST}
      - TEMPORAL_PORT=${TEMPORAL_PORT}
      - TEMPORAL_NAMESPACE=${TEMPORAL_NAMESPACE}
      - A2A_PORT=8080
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - LOG_LEVEL=info
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M
          cpus: '0.25'
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes --maxmemory 1gb --maxmemory-policy allkeys-lru
    deploy:
      replicas: 1
      resources:
        limits:
          memory: 1G
          cpus: '0.5'
```

#### Building Production Image
```dockerfile
# Multi-stage build for minimal production image
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY gateway/go.mod gateway/go.sum ./
RUN go mod download

COPY gateway/ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

COPY --from=builder /app/gateway .
COPY --from=builder /app/config ./config

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["./gateway"]
```

### Kubernetes Deployment

#### Namespace and ConfigMap
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: a2a-gateway

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: a2a-gateway-config
  namespace: a2a-gateway
data:
  TEMPORAL_HOST: "temporal.temporal-system.svc.cluster.local"
  TEMPORAL_PORT: "7233"
  TEMPORAL_NAMESPACE: "production"
  A2A_PORT: "8080"
  REDIS_URL: "redis://redis.cache.svc.cluster.local:6379"
  LOG_LEVEL: "info"
  AGENT_ROUTING_CONFIG: "/config/agent-routing.yaml"

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: agent-routing-config
  namespace: a2a-gateway
data:
  agent-routing.yaml: |
    version: "1.0"
    routing:
      "echo-agent":
        taskQueue: "echo-agent-tasks"
        workflowType: "EchoTaskWorkflow"
      "custom-agent":
        taskQueue: "custom-agent-tasks"
        workflowType: "LLMAgentWorkflow"
```

#### Secret Management
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: a2a-gateway-secrets
  namespace: a2a-gateway
type: Opaque
data:
  JWT_SECRET: <base64-encoded-jwt-secret>
```

#### Deployment Manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: a2a-gateway
  namespace: a2a-gateway
  labels:
    app: a2a-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: a2a-gateway
  template:
    metadata:
      labels:
        app: a2a-gateway
    spec:
      containers:
      - name: gateway
        image: temporal-a2a-gateway:latest
        ports:
        - containerPort: 8080
          name: http
        envFrom:
        - configMapRef:
            name: a2a-gateway-config
        - secretRef:
            name: a2a-gateway-secrets
        volumeMounts:
        - name: config-volume
          mountPath: /config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
      volumes:
      - name: config-volume
        configMap:
          name: agent-routing-config

---
apiVersion: v1
kind: Service
metadata:
  name: a2a-gateway-service
  namespace: a2a-gateway
spec:
  selector:
    app: a2a-gateway
  ports:
  - name: http
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

#### Horizontal Pod Autoscaler
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: a2a-gateway-hpa
  namespace: a2a-gateway
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: a2a-gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Helm Chart Deployment

#### values.yaml
```yaml
replicaCount: 3

image:
  repository: temporal-a2a-gateway
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: LoadBalancer
  port: 80
  targetPort: 8080

ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: a2a-gateway.example.com
      paths:
        - path: /
          pathType: Prefix

config:
  temporal:
    host: temporal.temporal-system.svc.cluster.local
    port: 7233
    namespace: production
  redis:
    url: redis://redis.cache.svc.cluster.local:6379
  logging:
    level: info

secrets:
  jwtSecret: your-production-jwt-secret

resources:
  requests:
    memory: 256Mi
    cpu: 250m
  limits:
    memory: 512Mi
    cpu: 500m

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
```

## Cloud Provider Specific Deployments

### AWS EKS Deployment

#### Prerequisites
```bash
# Install AWS CLI and kubectl
aws configure
eksctl create cluster --name a2a-gateway --region us-west-2

# Configure kubectl
aws eks update-kubeconfig --name a2a-gateway --region us-west-2
```

#### EKS-specific Configuration
```yaml
# Use AWS Load Balancer Controller
apiVersion: v1
kind: Service
metadata:
  name: a2a-gateway-service
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: a2a-gateway
```

### Google GKE Deployment

#### Prerequisites
```bash
# Create GKE cluster
gcloud container clusters create a2a-gateway \
  --num-nodes=3 \
  --enable-autoscaling \
  --min-nodes=2 \
  --max-nodes=10 \
  --region=us-central1
```

#### GKE-specific Configuration
```yaml
# Use Google Cloud Load Balancer
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: a2a-gateway-ssl-cert
spec:
  domains:
    - a2a-gateway.example.com

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: a2a-gateway-ingress
  annotations:
    kubernetes.io/ingress.global-static-ip-name: a2a-gateway-ip
    networking.gke.io/managed-certificates: a2a-gateway-ssl-cert
spec:
  rules:
  - host: a2a-gateway.example.com
    http:
      paths:
      - path: /*
        pathType: ImplementationSpecific
        backend:
          service:
            name: a2a-gateway-service
            port:
              number: 80
```

## Monitoring and Observability

### Prometheus Monitoring
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: a2a-gateway-metrics
spec:
  selector:
    matchLabels:
      app: a2a-gateway
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### Grafana Dashboard
Key metrics to monitor:
- Request latency (P50, P95, P99)
- Task creation rate
- Task completion rate
- Error rate by category
- Active task count
- Resource utilization

### Logging Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
data:
  fluent-bit.conf: |
    [INPUT]
        Name tail
        Path /var/log/containers/*a2a-gateway*.log
        Parser docker
        Tag kube.*
        
    [OUTPUT]
        Name es
        Match kube.*
        Host elasticsearch.logging.svc.cluster.local
        Port 9200
        Index a2a-gateway-logs
```

## Security Considerations

### Network Security
- Use Kubernetes NetworkPolicies to restrict pod-to-pod communication
- Enable TLS/mTLS for all service communication
- Implement proper ingress security (WAF, rate limiting)

### Secret Management
- Use external secret management (AWS Secrets Manager, HashiCorp Vault)
- Rotate JWT secrets regularly
- Implement proper RBAC for secret access

### Container Security
- Use minimal base images (alpine, distroless)
- Run containers as non-root user
- Implement resource limits and security contexts
- Regular vulnerability scanning

## Backup and Disaster Recovery

### Data Persistence
- Redis data backup strategy
- Temporal workflow state persistence
- Configuration backup procedures

### High Availability
- Multi-zone deployment
- Database clustering
- Circuit breakers and failover mechanisms

## Performance Tuning

### Resource Allocation
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

### Scaling Strategies
- Horizontal Pod Autoscaler based on CPU/memory
- Vertical Pod Autoscaler for right-sizing
- Cluster autoscaler for node scaling

### Optimization Checklist
- [ ] Enable connection pooling for Temporal
- [ ] Configure Redis memory policies
- [ ] Implement request batching where appropriate
- [ ] Use CDN for static assets
- [ ] Enable compression for HTTP responses

## Troubleshooting

### Common Deployment Issues

#### Pod Startup Failures
1. Check resource limits and requests
2. Verify environment variable configuration
3. Ensure external dependencies are accessible
4. Review security contexts and permissions

#### Service Discovery Issues
1. Verify DNS resolution within cluster
2. Check service and endpoint configurations
3. Test network connectivity between services
4. Review ingress and load balancer settings

#### Performance Problems
1. Monitor resource utilization
2. Check for resource contention
3. Review garbage collection logs
4. Analyze network latency between services

### Health Check Endpoints
- Gateway health: `/health`
- Metrics: `/metrics`
- Error codes: `/errors`

Use these endpoints for monitoring and troubleshooting deployment issues.