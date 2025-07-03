# Temporal A2A Gateway

A production-ready implementation of the Agent-to-Agent (A2A) protocol using Temporal for reliable agent orchestration. This implementation provides enterprise-grade durability, scalability, and observability for multi-agent systems.

## Overview

The Temporal A2A Gateway implements the [A2A Protocol specification](https://a2aproject.github.io/A2A/latest/) with Temporal workflows as the core orchestration engine. This design ensures that agent interactions are reliable, durable, and can be monitored in production environments.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                               Client Layer                                          │
├─────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │  Google A2A SDK │  │  Custom Client  │  │   Web UI/CLI    │  │  Third-party    │  │
│ │                 │  │                 │  │                 │  │    SDKs         │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────┬───────────────────────────────────────────────────────────────┘
                      │ HTTP/JSON-RPC 2.0
                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                            A2A Gateway (Go)                                         │
├─────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │   JSON-RPC 2.0  │  │  Agent Routing  │  │ Health Monitor  │  │   Telemetry     │  │
│ │    Handler      │  │   Configuration │  │   & Metrics     │  │   (OpenTel)     │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │ A2A Protocol    │  │   Task State    │  │  Error Handler  │  │ Agent Discovery │  │
│ │   Validation    │  │   Management    │  │   & Timeouts    │  │   & Registry    │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│ ┌─────────────────┐                                                                 │
│ │  Redis Cache    │                                                                 │
│ │   Interface     │                                                                 │
│ └─────────────────┘                                                                 │
└─────────────────────┬───────────────────────────┬─────────────┬----─────────────────┘
                      │ Temporal Client           │ Redis       │ HTTP/REST
                      ▼                           ▼             ▼
┌───────────────────────────────────┐ ┌─────────────────┐ ┌─────────────────────────────────┐
│         Temporal Cluster          │ │   Redis Cache   │ │        Agent Registry           │
├───────────────────────────────────┤ │                 │ ├─────────────────────────────────┤
│ ┌─────────────────┐ ┌───────────┐ │ │ • Task Metadata │ │ ┌─────────────┐ ┌─────────────┐ │
│ │ Temporal Server │ │  Web UI   │ │ │ • State Cache   │ │ │   Registry  │ │   Qdrant    │ │
│ │   (gRPC 7233)   │ │  (8233)   │ │ │ • Result Store  │ │ │ Service     │ │  Vector DB  │ │
│ └─────────────────┘ └───────────┘ │ │ • TTL Management│ │ │  (8001)     │ │   (6333)    │ │
│ ┌─────────────────┐ ┌───────────┐ │ └─────────────────┘ │ └─────────────┘ └─────────────┘ │
│ │ Workflow Engine │ │PostgreSQL │ │                     │ ┌─────────────┐ ┌─────────────┐ │
│ │                 │ │ Database  │ │                     │ │ Agent Cards │ │ Embeddings  │ │
│ └─────────────────┘ └───────────┘ │                     │ │  Storage    │ │  Search     │ │
│ ┌─────────────────┐ ┌───────────┐ │                     │ └─────────────┘ └─────────────┘ │
│ │  Task Queues    │ │Event Hist.│ │                     │ ┌─────────────┐ ┌─────────────┐ │
│ │                 │ │Persistence│ │                     │ │ Discovery   │ │ Capability  │ │
│ └─────────────────┘ └───────────┘ │                     │ │   Engine    │ │  Matching   │ │
└───────────────────────────────────┘                     └─────────────────────────────────┘
                      │ Temporal Worker Protocol
                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Agent Workers                                          │
├─────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │  Echo Worker    │  │   LLM Worker    │  │  Custom Worker  │  │  Tool Worker    │  │
│ │   (Python)      │  │   (Python)      │  │     (Any)       │  │   (Python)      │  │
│ │                 │  │                 │  │                 │  │                 │  │
│ │ • Echo Tasks    │  │ • LLM Inference │  │ • Custom Logic  │  │ • API Calls     │  │
│ │ • Simple I/O    │  │ • Prompt Mgmt   │  │ • Domain Specific│ │ • External Sys  │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │ Temporal Client │  │ Workflow Defns  │  │ Activity Funcs  │  │ Error Handling  │  │
│ │   (Python)      │  │   (@workflow)   │  │   (@activity)   │  │   & Retries     │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           External Services (Optional)                              │
├─────────────────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│ │   Monitoring    │  │   Log Aggreg.   │  │  External APIs  │  │   Third-party   │  │
│ │ (Prometheus)    │  │   (Jaeger)      │  │   (Various)     │  │   Integrations  │  │
│ │                 │  │                 │  │                 │  │                 │  │
│ │ • Metrics       │  │ • Traces        │  │ • LLM Services  │  │ • Custom Tools  │  │
│ │ • Alerting      │  │ • Debugging     │  │ • External DBs  │  │ • Extensions    │  │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Architectural Principles:**

1. **Protocol Compliance**: Full A2A Protocol v0.2.5 + JSON-RPC 2.0 adherence
2. **Reliability**: Temporal ensures durable execution with automatic retries
3. **Scalability**: Horizontal scaling via worker pools and task queue distribution  
4. **Observability**: OpenTelemetry integration for metrics, traces, and monitoring
5. **Flexibility**: Agent routing configuration allows diverse worker implementations
6. **Performance**: Redis caching layer for fast task state access

## Key Features

### Core A2A Protocol Support
- Complete JSON-RPC 2.0 A2A protocol implementation
- Agent registration and discovery
- Task creation, monitoring, and cancellation
- Message passing between agents
- Standardized error handling

### Temporal Integration
- Durable task execution with automatic retries
- Workflow state persistence
- Distributed agent coordination
- Built-in timeout and cancellation handling

### Production Features
- OpenTelemetry metrics and distributed tracing
- Redis-based task caching and indexing
- Comprehensive error classification system
- Health monitoring endpoints
- Environment validation

### Implementation Extensions
- Agent routing configuration via YAML
- Task metadata indexing and querying
- Automatic task cleanup and lifecycle management
- Custom agent capability definitions

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.24+ (for development)
- Python 3.12+ (for examples)

### Environment Configuration

For custom deployments, copy the example environment file and configure:

```bash
cp .env.example .env
# Edit .env with your specific configuration
```

**Required Environment Variables:**
- `TEMPORAL_HOST` - Temporal server hostname
- `TEMPORAL_PORT` - Temporal server port (default: 7233)
- `A2A_PORT` - Gateway HTTP port (default: 8080)
- `JWT_SECRET` - Secret for JWT token signing (generate with: `openssl rand -base64 32`)
- `OPENAI_API_KEY` - OpenAI API key (required for LLM workers)

**Key Optional Variables:**
- `REDIS_URL` - Redis connection string (improves performance)
- `AGENT_REGISTRY_URL` - Agent discovery service URL
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)

See `.env.example` for complete configuration options including LLM providers, monitoring, and worker settings.

### Running the Gateway

1. Start the complete stack:
```bash
docker-compose -f examples/docker-compose.yml up
```

2. Verify the gateway is running:
```bash
curl http://localhost:8080/health
```

3. Run the Google A2A SDK integration example:
```bash
cd examples/python
pip install -r requirements.txt
python google_a2a_sdk_integration_example.py
```

## Documentation

- [Implementation Guide](docs/implementation.md) - Technical implementation details
- [Configuration Reference](docs/configuration.md) - Gateway and worker configuration
- [API Reference](docs/api.md) - Complete API documentation
- [Deployment Guide](docs/deployment.md) - Production deployment instructions
- [Extension Guide](docs/extensions.md) - Implementation-specific extensions

## License

Apache License 2.0
