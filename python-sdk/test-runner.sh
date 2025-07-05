#!/bin/bash
set -e

echo "🧪 Running Temporal A2A SDK Tests"
echo "================================"

# Wait for gateway to be available (for integration tests)
echo "⏳ Waiting for gateway to be available..."
timeout 30 bash -c 'until curl -f ${GATEWAY_URL:-http://localhost:8080}/health 2>/dev/null; do sleep 1; done' || echo "⚠️  Gateway not available - integration tests may fail"

echo ""
echo "🔧 Running Unit Tests..."
echo "------------------------"
python -m pytest tests/test_sdk_patterns.py -v --tb=short

echo ""
echo "🌐 Running Integration Tests..."
echo "------------------------------"
python -m pytest tests/test_sdk_integration.py -v --tb=short

echo ""
echo "📊 Running All Tests with Coverage..."
echo "------------------------------------"
python -m pytest tests/ -v --tb=short --durations=10

echo ""
echo "✅ All tests completed!"