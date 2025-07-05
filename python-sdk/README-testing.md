# SDK Testing Guide

## Running Tests

### Using Docker Compose (Recommended)

Run tests in isolated environment:

```bash
# Start all services and run tests
cd examples/
docker-compose --profile tests up --build sdk-tests

# Or run tests after services are already running
docker-compose run --rm sdk-tests
```

### Manual Testing

If you prefer to run tests locally:

```bash
cd python-sdk/

# Install test dependencies
pip install -r requirements-test.txt

# Run unit tests only
python -m pytest tests/test_sdk_patterns.py -v

# Run integration tests (requires running gateway)
GATEWAY_URL=http://localhost:8080 python -m pytest tests/test_sdk_integration.py -v

# Run all tests
python -m pytest tests/ -v
```

## Test Categories

### Unit Tests (`test_sdk_patterns.py`)
- **@agent_activity Pattern**: Decorator functionality
- **Agent Class Structure**: Agent instantiation and configuration
- **Activity Collection**: Automatic discovery of decorated methods
- **Streaming Patterns**: Stream parameter injection
- **Code Reduction**: Verify 85%+ reduction achieved
- **Pure Business Logic**: Zero framework dependencies
- **Memory Efficiency**: O(1) streaming verification
- **Clean Separation**: temporal.agent vs temporal.a2a imports

### Integration Tests (`test_sdk_integration.py`)
- **End-to-End Echo**: Full workflow via gateway
- **Progressive Streaming**: Word-by-word delivery validation
- **A2A Protocol Compliance**: Proper artifact structure
- **Code Reduction Metrics**: Line count verification

## Test Environment

The test container includes:
- Python 3.11
- pytest with async support
- httpx for HTTP testing
- temporalio SDK
- Full SDK installation in development mode

## Health Checks

Tests automatically wait for:
- Gateway availability (`/health` endpoint)
- All agent workers to be running
- Temporal server connectivity

## Expected Results

✅ **All tests should pass** when the SDK is working correctly:
- Echo agent responds with "Echo: [message]"
- Streaming delivers progressive chunks
- 85%+ code reduction achieved
- Clean import separation maintained
- A2A v0.2.5 protocol compliance verified