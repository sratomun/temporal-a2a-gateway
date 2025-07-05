# Webhook-Based Real-Time Streaming Architecture

## Overview

This document describes a webhook-based architecture that enables true real-time streaming from Temporal activities to clients, solving the fundamental limitation that activities must complete before returning data.

## Problem Statement

### Current Limitation
- Temporal activities cannot return data until they complete
- Activities cannot send signals directly to workflows
- Current implementation collects all chunks before returning (batch mode)
- Real-time streaming requires external dependencies (Redis/Kafka)

### Current Flow (Batch Mode)
```
Activity generates chunks → Collects in memory → Activity completes → Returns all chunks → Workflow processes → Signals gateway → SSE to client
```

## Proposed Solution: Webhook-Based Streaming

### Architecture Overview
```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│  Agent Activity │  HTTP   │     Gateway     │   SSE   │     Client      │
│   (running)     │ ──────> │ Webhook Handler │ ──────> │   (Browser)     │
└─────────────────┘  POST   └─────────────────┘  Push   └─────────────────┘
     ↓                             ↓
  Generates                 Routes by task_id
   chunks                   to active stream
```

### Key Components

#### 1. Gateway Webhook Endpoint
A new internal endpoint that receives chunks from running activities:

```go
// POST /internal/webhook/stream
type StreamChunkRequest struct {
    TaskID      string `json:"task_id"`
    Chunk       string `json:"chunk"`
    Sequence    int    `json:"sequence"`
    IsLast      bool   `json:"is_last"`
    ArtifactID  string `json:"artifact_id"`
}

func (g *Gateway) handleStreamWebhook(w http.ResponseWriter, r *http.Request) {
    var req StreamChunkRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Find active SSE connection for this task
    if stream, exists := g.activeStreams[req.TaskID]; exists {
        event := SSEvent{
            Event: "artifact-update",
            Data: ArtifactUpdate{
                ArtifactID: req.ArtifactID,
                Parts: []Part{{Kind: "text", Text: req.Chunk}},
                Append: req.Sequence > 1,
                LastChunk: req.IsLast,
            },
        }
        stream.Send(event)
    }
}
```

#### 2. Enhanced StreamingContext
Modified to send chunks via webhook instead of collecting:

```python
class StreamingContext:
    """Real-time streaming via gateway webhooks"""
    
    def __init__(self, message_data: Dict[str, Any]):
        self.task_id = activity.info().workflow_id
        self.chunk_count = 0
        self.artifact_id = f"artifact-{self.task_id[:8]}"
        self.gateway_url = os.getenv(
            "GATEWAY_WEBHOOK_URL", 
            "http://gateway:8080/internal/webhook/stream"
        )
        # Fallback for network failures
        self.fallback_chunks = []
    
    async def send_chunk(self, chunk: str) -> None:
        """Send chunk immediately via webhook"""
        self.chunk_count += 1
        
        payload = {
            "task_id": self.task_id,
            "chunk": chunk,
            "sequence": self.chunk_count,
            "is_last": False,
            "artifact_id": self.artifact_id
        }
        
        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    self.gateway_url,
                    json=payload,
                    headers=self._auth_headers(),
                    timeout=aiohttp.ClientTimeout(total=5)
                ) as resp:
                    if resp.status != 200:
                        raise Exception(f"Webhook failed: {resp.status}")
                        
            logger.info(f"📤 Streamed chunk {self.chunk_count} in real-time")
            
        except Exception as e:
            logger.warning(f"Webhook failed, falling back to batch: {e}")
            self.fallback_chunks.append(chunk)
    
    async def finish(self) -> None:
        """Send final chunk marker"""
        if not self.fallback_chunks:  # Only if streaming worked
            await self._send_final_marker()
    
    def _auth_headers(self) -> Dict[str, str]:
        """Internal authentication headers"""
        return {
            "X-Internal-Auth": self._generate_hmac(),
            "X-Task-ID": self.task_id,
            "X-From": "temporal-activity"
        }
```

#### 3. Activity Implementation (Unchanged)
The beauty is that activity code remains the same:

```python
@agent_activity
async def process_streaming_activity(self, text: str, stream) -> None:
    """Activity that streams in real-time"""
    async for chunk in generate_chunks(text):
        await stream.send_chunk(chunk)  # Now real-time via webhook!
        await asyncio.sleep(0.5)
    
    await stream.finish()
```

### Security Considerations

#### Internal Authentication
- Webhook endpoint only accessible within cluster network
- HMAC-based request signing
- Task ID validation against active streams

#### Network Security
```yaml
# Docker network isolation
services:
  gateway:
    networks:
      - internal  # Not exposed externally
    ports:
      - "8080:8080"  # Public API
      # Webhook port not exposed
  
  workers:
    networks:
      - internal  # Same network as gateway
```

### Failure Handling

#### Graceful Degradation
1. **Network failures**: Fall back to batch collection
2. **Gateway unavailable**: Collect chunks and return at end
3. **No active stream**: Log and continue (client polling will get result)

```python
async def send_chunk(self, chunk: str):
    """With retry and fallback"""
    for attempt in range(3):
        try:
            await self._webhook_post(chunk)
            return  # Success
        except Exception as e:
            if attempt == 2:  # Final attempt
                logger.error(f"Webhook failed, using fallback: {e}")
                self.fallback_chunks.append(chunk)
                self.mode = "batch"  # Switch to batch mode
```

### Performance Characteristics

#### Latency
- **Current (Batch)**: Activity duration + workflow processing
- **Webhook-based**: ~10-50ms per chunk (network hop)

#### Throughput
- Supports hundreds of concurrent streams
- Limited by gateway's SSE connection handling
- Each chunk is an independent HTTP request

#### Resource Usage
- No message queues or external storage
- Minimal memory (no chunk accumulation)
- Network traffic: One HTTP request per chunk

### Implementation Phases

#### Phase 1: Basic Webhook Streaming
- Add webhook endpoint to gateway
- Modify StreamingContext for webhook calls
- Test with echo streaming worker

#### Phase 2: Production Hardening
- Add authentication and validation
- Implement retry logic
- Add metrics and monitoring

#### Phase 3: Advanced Features
- Chunk batching for high-frequency streams
- Compression for large chunks
- WebSocket alternative for bi-directional needs

### Comparison with Alternatives

| Approach | Real-time | Complexity | Dependencies | Reliability |
|----------|-----------|------------|--------------|-------------|
| Current (Batch) | ❌ | Low | None | High |
| Redis Queue | ✅ | Medium | Redis | High |
| Kafka Streaming | ✅ | High | Kafka | Very High |
| **Webhook** | ✅ | Low | None | Medium-High |

### Advantages

1. **Simplicity**: Just HTTP calls between services
2. **No Dependencies**: No Redis, Kafka, or message queues
3. **Real-time**: Chunks stream as generated
4. **Compatibility**: Works with existing activity patterns
5. **Debuggable**: Simple HTTP requests easy to trace

### Limitations

1. **Network Reliability**: Requires stable internal network
2. **Gateway Availability**: Gateway must be running
3. **No Persistence**: Chunks not stored if stream disconnects
4. **Unidirectional**: Activity → Client only

### Future Enhancements

1. **WebSocket Support**: For bidirectional streaming
2. **Chunk Buffering**: Small buffer for network hiccups
3. **Stream Multiplexing**: Multiple clients per task
4. **Compression**: For bandwidth optimization

## Scalability Analysis

### Overview
Understanding the scalability characteristics of the webhook approach is crucial for production deployment.

### Request Volume Calculations
```
Single Stream:
- 10 chunks/second = 10 HTTP requests/second
- 100 bytes/chunk + 200 bytes HTTP overhead = 300 bytes/request
- Total: 3 KB/second per stream

At Scale:
- 1,000 concurrent streams = 10,000 requests/second = 3 MB/second
- 10,000 concurrent streams = 100,000 requests/second = 30 MB/second  
- 100,000 concurrent streams = 1,000,000 requests/second = 300 MB/second
```

### Scalability Bottlenecks

#### 1. HTTP Connection Overhead
**Problem**: Each chunk creates a new HTTP request

**Solutions**:
```python
# Connection pooling
class StreamingContext:
    _session = None  # Shared across all activities
    
    @classmethod
    async def get_session(cls):
        if not cls._session:
            connector = aiohttp.TCPConnector(
                limit=1000,  # Total connection pool
                limit_per_host=100,  # Per gateway instance
                keepalive_timeout=30
            )
            cls._session = aiohttp.ClientSession(
                connector=connector,
                timeout=aiohttp.ClientTimeout(total=5)
            )
        return cls._session
```

#### 2. Gateway Processing Capacity
**Problem**: Single gateway becomes bottleneck

**Solutions**:

a) **Horizontal Scaling**
```yaml
# Multiple gateway instances behind load balancer
services:
  gateway-1:
    image: a2a-gateway
    deploy:
      replicas: 3
  
  nginx:
    image: nginx
    configs:
      - source: nginx-config
        target: /etc/nginx/nginx.conf
```

b) **Efficient Request Processing**
```go
// Pre-allocated object pools
var chunkPool = sync.Pool{
    New: func() interface{} {
        return new(StreamChunkRequest)
    },
}

// Lock-free concurrent map for stream routing
type StreamRouter struct {
    streams sync.Map  // No mutex needed
}
```

#### 3. Network Bandwidth
**At different scales**:
- 1K streams: 24 Mbps - ✅ Single gateway sufficient
- 10K streams: 240 Mbps - ⚠️ May need multiple gateways
- 100K streams: 2.4 Gbps - ❌ Need distributed architecture

### Optimization Strategies

#### 1. Smart Batching
For high-frequency streams, batch multiple chunks:

```python
class BatchingStreamContext:
    def __init__(self):
        self.batch = []
        self.batch_size = 10
        self.batch_timeout = 0.1  # 100ms
        self.batch_timer = None
        
    async def send_chunk(self, chunk: str):
        self.batch.append(chunk)
        
        if len(self.batch) >= self.batch_size:
            await self._flush_batch()
        elif not self.batch_timer:
            self.batch_timer = asyncio.create_task(self._delayed_flush())
    
    async def _flush_batch(self):
        if self.batch:
            await self._webhook_post({
                "chunks": self.batch,
                "count": len(self.batch)
            })
            self.batch = []
```

#### 2. HTTP/2 Multiplexing
Reduce connection overhead with HTTP/2:

```python
# Single connection, multiple streams
connector = aiohttp.TCPConnector(
    force_close=False,
    enable_cleanup_closed=True,
    # HTTP/2 support (if available)
)
```

#### 3. Per-Workflow Webhooks
**Concept**: Each workflow gets unique webhook endpoint

**Pros**:
- Direct routing without lookup
- Better isolation
- Easier debugging

**Cons**:
- Dynamic endpoint management
- Not significantly better than task_id in payload

**Verdict**: Not necessary for most use cases

### Scale Recommendations

#### < 1,000 Concurrent Streams
✅ **Webhooks work perfectly**
- Single gateway instance
- Basic implementation
- No optimizations needed

#### 1,000 - 10,000 Concurrent Streams
⚠️ **Webhooks with optimizations**
- Implement connection pooling
- Consider batching for high-frequency streams
- Monitor gateway CPU/memory
- Prepare for horizontal scaling

#### 10,000 - 100,000 Concurrent Streams
🔧 **Advanced optimizations required**
- Multiple gateway instances
- HTTP/2 or gRPC
- Smart batching mandatory
- Consider hybrid approach (webhooks + queue for overflow)

#### > 100,000 Concurrent Streams
❌ **Alternative architecture needed**
- Redis Pub/Sub for fan-out
- Kafka for durability
- Direct gRPC streaming
- WebSocket connections

### Comparison with Message Queue Approaches

| Metric | Webhooks | Redis Pub/Sub | Kafka | Direct gRPC |
|--------|----------|---------------|-------|-------------|
| **Complexity** | Low | Medium | High | Medium |
| **Latency** | 10-50ms | 1-5ms | 10-50ms | <1ms |
| **Throughput** | 100K/sec* | 1M/sec | 10M/sec | 500K/sec |
| **Dependencies** | None | Redis | Kafka cluster | None |
| **Durability** | No | Limited | Yes | No |
| **Best for** | <10K streams | 10K-100K | >100K | Real-time |

*With optimizations

### Production Deployment Considerations

#### 1. Monitoring
```python
# Add metrics
async def send_chunk(self, chunk: str):
    start_time = time.time()
    try:
        await self._webhook_post(chunk)
        metrics.webhook_latency.observe(time.time() - start_time)
        metrics.webhook_success.inc()
    except Exception as e:
        metrics.webhook_failure.inc()
        raise
```

#### 2. Circuit Breaker
```python
class CircuitBreaker:
    def __init__(self, failure_threshold=5, timeout=60):
        self.failure_count = 0
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.last_failure_time = None
        self.is_open = False
```

#### 3. Graceful Degradation
```python
async def send_chunk_with_fallback(self, chunk: str):
    if self.webhook_healthy:
        try:
            await self._webhook_post(chunk)
            return
        except Exception:
            self.webhook_healthy = False
    
    # Fallback to batch mode
    self.fallback_chunks.append(chunk)
```

## gRPC Alternative: Activity-to-Gateway Streaming

### Architecture Overview
Instead of HTTP webhooks, use gRPC streaming for true bidirectional communication:

```proto
// streaming.proto
service StreamingGateway {
    // Bidirectional streaming for real-time chunks
    rpc StreamChunks(stream ChunkRequest) returns (stream ChunkResponse);
    
    // Server streaming for one-way chunk delivery
    rpc SendChunks(stream ChunkRequest) returns (ChunkAck);
}

message ChunkRequest {
    string task_id = 1;
    string artifact_id = 2;
    string content = 3;
    int32 sequence = 4;
    bool is_last = 5;
}
```

### gRPC Implementation

#### Activity Side (Python)
```python
import grpc
from concurrent import futures
import streaming_pb2_grpc as pb2_grpc

class GRPCStreamingContext:
    """gRPC-based streaming from activities"""
    
    def __init__(self, message_data: Dict[str, Any]):
        self.task_id = activity.info().workflow_id
        self.channel = None
        self.stub = None
        self.stream = None
        self._connect()
    
    def _connect(self):
        """Establish gRPC connection with keep-alive"""
        options = [
            ('grpc.keepalive_time_ms', 10000),
            ('grpc.keepalive_timeout_ms', 5000),
            ('grpc.keepalive_permit_without_calls', True),
            ('grpc.http2.max_pings_without_data', 0)
        ]
        
        self.channel = grpc.insecure_channel(
            'gateway:50051',  # gRPC port
            options=options
        )
        self.stub = pb2_grpc.StreamingGatewayStub(self.channel)
    
    async def send_chunk(self, chunk: str) -> None:
        """Send chunk via gRPC stream"""
        request = ChunkRequest(
            task_id=self.task_id,
            artifact_id=self.artifact_id,
            content=chunk,
            sequence=self.chunk_count,
            is_last=False
        )
        
        # Use generator for streaming
        def request_generator():
            yield request
        
        # Server streaming RPC (more efficient than bidirectional)
        response = self.stub.SendChunks(request_generator())
        
        if not response.acknowledged:
            raise Exception("Chunk not acknowledged")
```

#### Gateway Side (Go)
```go
// Gateway gRPC server
type streamingServer struct {
    pb.UnimplementedStreamingGatewayServer
    activeStreams map[string]*SSEStream
    mu sync.RWMutex
}

func (s *streamingServer) SendChunks(stream pb.StreamingGateway_SendChunksServer) error {
    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&pb.ChunkAck{Acknowledged: true})
        }
        if err != nil {
            return err
        }
        
        // Route to SSE stream
        s.mu.RLock()
        if sseStream, exists := s.activeStreams[chunk.TaskId]; exists {
            event := createSSEEvent(chunk)
            sseStream.Send(event)
        }
        s.mu.RUnlock()
    }
}
```

### Performance Comparison: gRPC vs HTTP Webhooks

| Metric | HTTP Webhooks | gRPC Streaming | Advantage |
|--------|--------------|----------------|-----------|
| **Connection Overhead** | New per chunk | Persistent stream | ✅ gRPC (10x better) |
| **Latency** | 10-50ms | 0.1-1ms | ✅ gRPC (50x better) |
| **Throughput** | 10K req/sec | 100K+ msg/sec | ✅ gRPC (10x better) |
| **Memory Usage** | Higher (HTTP headers) | Lower (binary) | ✅ gRPC |
| **CPU Usage** | Higher (HTTP parsing) | Lower (protobuf) | ✅ gRPC |
| **Network Efficiency** | 300 bytes/chunk | 50 bytes/chunk | ✅ gRPC (6x better) |

### Scalability Analysis

#### Connection Multiplexing
```
HTTP Webhooks:
- 1,000 streams = 1,000+ TCP connections (with pooling)
- Connection setup/teardown overhead
- HTTP headers repeated each request

gRPC:
- 1,000 streams = 10-100 TCP connections (HTTP/2 multiplexing)
- Single connection handles multiple streams
- Binary protocol, minimal overhead
```

#### At Scale Performance
```
10K concurrent streams:
- HTTP: 100K req/sec, 30 MB/sec bandwidth
- gRPC: 100K msg/sec, 5 MB/sec bandwidth (6x more efficient)

100K concurrent streams:
- HTTP: Requires load balancing, connection exhaustion issues
- gRPC: Handles on fewer servers due to multiplexing

1M concurrent streams:
- HTTP: Not feasible without major architecture changes
- gRPC: Possible with proper stream management
```

### Implementation Complexity

#### Development Complexity
- **HTTP**: ✅ Simple (just HTTP POST)
- **gRPC**: ⚠️ Medium (protobuf, code generation, streaming patterns)

#### Operational Complexity
- **HTTP**: ✅ Simple (standard HTTP debugging)
- **gRPC**: ⚠️ Medium (specialized tools, binary protocol)

### Recommendation: Hybrid Approach

#### Sprint 5: Start with HTTP Webhooks
- Faster to implement
- Easier to debug
- Proves the concept

#### Sprint 6+: Add gRPC for Scale
```python
class StreamingContext:
    def __init__(self):
        self.mode = self._determine_mode()
    
    def _determine_mode(self):
        if os.getenv("STREAMING_MODE") == "grpc":
            return GRPCStreamingContext()
        else:
            return HTTPWebhookContext()
```

### When to Use Each Approach

**HTTP Webhooks** (< 10K streams):
- Quick implementation needed
- Standard tooling preferred
- Moderate scale requirements

**gRPC Streaming** (> 10K streams):
- High performance critical
- Network efficiency matters
- Scale is primary concern

**Message Queues** (> 100K streams):
- Durability required
- Fan-out to multiple consumers
- Decoupling more important than latency

## Conclusion

While gRPC offers superior performance and scalability, HTTP webhooks provide a simpler path to real-time streaming. The recommendation is to start with webhooks for Sprint 5 and consider gRPC as an optimization when scale demands it.

**Key Insight**: gRPC scales 10x better than HTTP webhooks, but webhooks are 10x simpler to implement. Choose based on your immediate needs and scale requirements.