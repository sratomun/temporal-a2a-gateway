# Configuration Reference

This document describes all configuration options for the Temporal A2A Gateway.

## Environment Variables

### Required Configuration

#### Temporal Configuration
- **TEMPORAL_HOST**: Temporal server hostname
  - Default: `localhost`
  - Example: `temporal.example.com`

- **TEMPORAL_PORT**: Temporal server port
  - Default: `7233`
  - Example: `7233`

- **TEMPORAL_NAMESPACE**: Temporal namespace
  - Default: `default`
  - Example: `production`

#### Gateway Configuration
- **A2A_PORT**: Port for the A2A Gateway to listen on
  - Default: `8080`
  - Example: `8080`

- **JWT_SECRET**: Secret key for JWT token signing (REQUIRED)
  - Generate with: `openssl rand -base64 32`
  - Example: `your-super-secret-jwt-signing-key-here`

#### LLM Provider Configuration
- **OPENAI_API_KEY**: OpenAI API key (required for LLM workers)
  - Example: `sk-your-openai-api-key-here`

- **OPENAI_MODEL**: OpenAI model to use
  - Default: `gpt-4`
  - Options: `gpt-4`, `gpt-3.5-turbo`, etc.

### Optional but Recommended

#### Redis Configuration
- **REDIS_URL**: Redis connection URL (improves performance)
  - Default: `redis://redis:6379`
  - Examples:
    - `redis://localhost:6379`
    - `rediss://user:pass@redis.example.com:6380/0`

#### Agent Registry Configuration
- **AGENT_REGISTRY_URL**: Agent registry service URL
  - Default: `http://agent-registry:8001`
  - Example: `https://registry.example.com`

### Optional Configuration

#### Additional LLM Providers
- **ANTHROPIC_API_KEY**: Anthropic API key (for Claude models)
  - Example: `your-anthropic-api-key-here`

- **GOOGLE_API_KEY**: Google AI API key (for Gemini models)
  - Example: `your-google-api-key-here`

- **AZURE_OPENAI_ENDPOINT**: Azure OpenAI endpoint
  - Example: `https://your-resource.openai.azure.com`

- **AZURE_OPENAI_API_KEY**: Azure OpenAI API key
  - Example: `your-azure-api-key-here`

#### Logging
- **LOG_LEVEL**: Logging verbosity level
  - Default: `info`
  - Options: `debug`, `info`, `warn`, `error`

#### Database
- **DATABASE_URL**: PostgreSQL connection URL (optional)
  - Default: None (not required for basic operation)
  - Example: `postgres://user:pass@localhost:5432/a2a_gateway`

#### Agent Routing
- **AGENT_ROUTING_CONFIG**: Path to agent routing configuration file
  - Default: `config/agent-routing.yaml`
  - Example: `/etc/a2a/routing.yaml`

## Agent Routing Configuration

### File Format
The agent routing configuration is defined in YAML format:

```yaml
version: "1.0"

routing:
  "Agent Name":
    taskQueue: "task-queue-name"
    workflowType: "WorkflowTypeName"

workflowCategories:
  CategoryName:
    description: "Category description"
    examples: ["Example 1", "Example 2"]
```

### Example Configuration
```yaml
version: "1.0"

routing:
  # Echo agent for testing
  "echo-agent":
    taskQueue: "echo-agent-tasks"
    workflowType: "EchoTaskWorkflow"
  
  # Example production agent (customize as needed)
  "custom-agent":
    taskQueue: "custom-agent-tasks"
    workflowType: "LLMAgentWorkflow"

workflowCategories:
  LLMAgentWorkflow:
    description: "For agents that use LLM for reasoning and generation"
    examples: ["Custom Agent", "LLM-based Agent"]
  
  ToolAgentWorkflow:
    description: "For agents that primarily execute tools and APIs"
    examples: ["Database Agent", "Deployment Agent"]
```

### Routing Rules
1. Agent names must exactly match the `agentId` field in task creation requests
2. Task queue names should follow the pattern: `{agent-name}-tasks`
3. Workflow types should be descriptive and categorized appropriately
4. All routing entries must specify both `taskQueue` and `workflowType`

## Docker Configuration

### Environment File
Copy and customize the provided example:

```bash
cp .env.example .env
# Edit .env with your configuration
```

Key sections in `.env`:

```bash
# Required Configuration
TEMPORAL_HOST=temporal
TEMPORAL_PORT=7233
A2A_PORT=8080
JWT_SECRET=your-production-jwt-secret-here
OPENAI_API_KEY=sk-your-openai-api-key-here

# Optional but Recommended
REDIS_URL=redis://redis:6379
AGENT_REGISTRY_URL=http://agent-registry:8001
LOG_LEVEL=info
```

### Docker Compose Override
For development, create `docker-compose.override.yml`:

```yaml
version: '3.8'

services:
  gateway:
    environment:
      - LOG_LEVEL=debug
    volumes:
      - ./config:/app/config
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics port

  temporal:
    ports:
      - "8233:8233"  # Temporal Web UI
```

## Kubernetes Configuration

### ConfigMap Example
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: a2a-gateway-config
data:
  TEMPORAL_HOST: "temporal.temporal-system.svc.cluster.local"
  TEMPORAL_PORT: "7233"
  TEMPORAL_NAMESPACE: "production"
  A2A_PORT: "8080"
  REDIS_URL: "redis://redis.cache.svc.cluster.local:6379"
  LOG_LEVEL: "info"
```

### Secret Example
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: a2a-gateway-secrets
type: Opaque
data:
  JWT_SECRET: base64-encoded-secret-here
```

## Validation Rules

The gateway performs comprehensive validation on startup:

### Required Variable Validation
- Missing required variables prevent startup
- Invalid URLs or ports cause startup failure
- Network connectivity to Temporal and Redis is verified

### Optional Variable Warnings
- Weak JWT secrets generate warnings
- Missing optional variables are logged but don't prevent startup

### Runtime Validation
- Agent routing configuration is validated on load
- YAML syntax errors prevent gateway startup
- Unknown workflow types generate warnings

## Performance Tuning

### Redis Configuration
For high-throughput deployments:

```bash
# Increase Redis memory limit
REDIS_MAXMEMORY=2gb
REDIS_MAXMEMORY_POLICY=allkeys-lru

# Enable persistence for task data
REDIS_SAVE=900 1 300 10 60 10000
```

### Temporal Configuration
For production workloads:

```bash
# Increase Temporal connection pool
TEMPORAL_CLIENT_POOL_SIZE=10

# Configure timeouts
TEMPORAL_CLIENT_TIMEOUT=30s
TEMPORAL_WORKFLOW_TIMEOUT=3600s
```

### Gateway Configuration
For high load:

```bash
# Increase worker pool size
GOMAXPROCS=8

# Configure garbage collection
GOGC=100
```

## Security Considerations

### Production Checklist
- [ ] Use strong, unique JWT secret
- [ ] Enable TLS/HTTPS for all external communication
- [ ] Implement proper authentication middleware
- [ ] Configure rate limiting
- [ ] Use Redis AUTH if Redis is network accessible
- [ ] Restrict network access to internal services
- [ ] Regular security updates for all dependencies

### Network Security
- Gateway should only be accessible from authorized clients
- Temporal and Redis should be on private networks
- Use network policies in Kubernetes environments
- Consider mTLS for service-to-service communication

## Monitoring Configuration

### Metrics Endpoints
- Gateway metrics: `http://gateway:8080/metrics`
- Health check: `http://gateway:8080/health`
- Error documentation: `http://gateway:8080/errors`

### Logging Configuration
```bash
# Structured logging
LOG_FORMAT=json

# Log sampling for high volume
LOG_SAMPLE_RATE=0.1

# Include trace IDs
LOG_INCLUDE_TRACE_ID=true
```

## Troubleshooting

### Common Configuration Issues

#### Gateway Won't Start
1. Check required environment variables
2. Verify Temporal connectivity
3. Validate YAML syntax in routing configuration
4. Check port availability

#### Tasks Not Routing
1. Verify agent name matches routing configuration exactly
2. Check task queue configuration
3. Ensure workers are running for the target queue
4. Review gateway logs for routing errors

#### Performance Issues
1. Monitor Redis memory usage
2. Check Temporal workflow queue depths
3. Review gateway metrics for bottlenecks
4. Verify adequate resources for all services