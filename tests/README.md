# Testing Guide

This directory contains comprehensive tests for the Temporal A2A Gateway implementation.

## Test Structure

```
tests/
├── unit/                    # Unit tests for individual components
│   └── gateway_test.go     # Gateway component unit tests
├── integration/            # Integration tests for A2A protocol compliance
│   └── a2a_protocol_test.py # A2A protocol compliance tests
├── e2e/                    # End-to-end tests (future)
└── README.md              # This file
```

## Prerequisites

### For Go Tests
- Go 1.24+
- Access to gateway source code

### For Python Integration Tests
- Python 3.12+
- requests library
- pytest (optional, recommended)
- Running gateway instance

## Running Tests

### Unit Tests

Run Go unit tests from the gateway directory:

```bash
cd gateway
go test ./...
```

Or run specific test files:

```bash
cd tests/unit
go test -v gateway_test.go
```

### Integration Tests

Start the gateway stack first:

```bash
# From project root
docker-compose -f examples/docker-compose.yml up -d

# Wait for services to be healthy
curl http://localhost:8080/health
```

Run integration tests:

```bash
cd tests/integration

# With pytest (recommended)
pip install pytest requests
python -m pytest a2a_protocol_test.py -v

# Or run directly
python a2a_protocol_test.py
```

### Environment Variables

Configure test environment:

```bash
# Gateway URL for integration tests
export A2A_GATEWAY_URL=http://localhost:8080

# Test timeout (seconds)
export TEST_TIMEOUT=30
```

## Test Categories

### Unit Tests (Go)

**Purpose**: Test individual components and functions in isolation.

**Coverage**:
- JSON-RPC parameter parsing
- AgentCard structure validation
- Capability configuration handling
- HTTP request processing
- Error handling

**Example**:
```bash
cd gateway
go test -v -run TestBasicAgentCardParsing
```

### Integration Tests (Python)

**Purpose**: Test A2A protocol compliance and real gateway interactions.

**Coverage**:
- Health endpoint functionality
- Agent listing
- Task creation and lifecycle
- Error handling and JSON-RPC compliance
- Concurrent task handling
- Large input processing

**Example**:
```bash
cd tests/integration
python -m pytest a2a_protocol_test.py::TestA2AProtocolCompliance::test_task_completion_flow -v
```

## Test Data and Fixtures

### Mock Agent Cards

Tests use various agent card configurations:

```go
// Basic agent card
AgentCard{
    Name:        "Test Agent",
    Description: "Test description",
    Version:     "1.0.0",
}

// Agent with capabilities
AgentCard{
    Name:        "Advanced Agent",
    Description: "Advanced test agent",
    Version:     "1.0.0",
    Capabilities: &AgentCapabilities{
        Streaming:              boolPtr(true),
        PushNotifications:      boolPtr(false),
        StateTransitionHistory: boolPtr(true),
    },
}
```

### Test Messages

Integration tests use A2A protocol message format:

```json
{
    "messages": [{
        "role": "user",
        "parts": [{
            "type": "text",
            "content": "Test message content"
        }]
    }]
}
```

## Continuous Integration

### GitHub Actions

Tests can be integrated into CI pipelines:

```yaml
name: Test Suite

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - run: cd gateway && go test ./...

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v4
        with:
          python-version: '3.13'
      - run: |
          docker-compose -f examples/docker-compose.yml up -d
          sleep 30  # Wait for services
          pip install pytest requests
          cd tests/integration && python -m pytest -v
```

## Test Development Guidelines

### Adding Unit Tests

1. Create test files with `_test.go` suffix
2. Place in appropriate subdirectory under `tests/unit/`
3. Follow Go testing conventions
4. Test both success and error cases
5. Use table-driven tests for multiple scenarios

### Adding Integration Tests

1. Add test methods to existing test classes
2. Use descriptive test names
3. Clean up resources after tests
4. Handle timeouts appropriately
5. Test error conditions

### Test Naming Conventions

- **Unit tests**: `TestFunctionName_Scenario`
- **Integration tests**: `test_feature_description`
- **Test files**: `component_test.go` or `feature_test.py`

## Troubleshooting

### Common Issues

**Gateway not responding**:
- Check if gateway is running: `curl http://localhost:8080/health`
- Verify Docker containers are up: `docker-compose ps`
- Check gateway logs: `docker-compose logs gateway`

**Tests timing out**:
- Increase `TEST_TIMEOUT` environment variable
- Check if echo-worker is running
- Verify Temporal connectivity

**Import errors in Python tests**:
- Ensure Python path includes examples directory
- Install required dependencies: `pip install -r requirements.txt`

**Go module issues**:
- Run `go mod tidy` in gateway directory
- Verify Go version: `go version`

### Debug Mode

Enable debug logging for more detailed test output:

```bash
# Gateway debug mode
export LOG_LEVEL=debug

# Python test debug
python -m pytest tests/integration/ -v -s --log-cli-level=DEBUG
```

## Performance Testing

While not included in this test suite, performance testing recommendations:

1. **Load Testing**: Use tools like `hey` or `wrk` to test gateway throughput
2. **Stress Testing**: Test concurrent task creation and completion
3. **Memory Testing**: Monitor memory usage during long-running tests
4. **Latency Testing**: Measure end-to-end task completion times

## Coverage Reports

### Go Coverage

```bash
cd gateway
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Python Coverage

```bash
pip install pytest-cov
cd tests/integration
python -m pytest --cov=../../examples/python --cov-report=html
```

## Contributing

When adding tests:

1. Ensure tests are independent and can run in any order
2. Use appropriate assertions and error messages
3. Document complex test scenarios
4. Update this README for new test categories
5. Verify tests pass in clean environment