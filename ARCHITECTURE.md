# Architecture Documentation

## System Overview

The Distributed Real-Time Push Notification Service is a high-throughput, fault-tolerant backend system designed to handle asynchronous push notifications for mobile applications at scale.

## Architecture Diagram

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Mobile Apps   │    │   Web Clients   │    │  External APIs  │
│   (iOS/Android) │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────┐
                    │   Load Balancer     │
                    │   (Future)          │
                    └─────────┬───────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │   API Gateway       │
                    │   (Port 8080)       │
                    │   - RESTful API     │
                    │   - Rate Limiting   │
                    │   - Request Validation │
                    └─────────┬───────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
                ▼             ▼             ▼
    ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
    │     Redis       │ │   Kafka     │ │   Monitoring    │
    │   (Cache)       │ │  (Queue)    │ │   & Logging     │
    │                 │ │             │ │                 │
    │ - User Sessions │ │ - Message   │ │ - Health Checks │
    │ - Rate Limiting │ │   Queue     │ │ - Metrics       │
    │ - Status Cache  │ │ - Durability│ │ - Alerting      │
    └─────────────────┘ └─────┬───────┘ └─────────────────┘
                              │
                              ▼
                    ┌─────────────────────┐
                    │   Consumer Service  │
                    │   (Multiple Workers)│
                    │                     │
                    │ - Message Processing│
                    │ - Retry Logic       │
                    │ - Platform Routing  │
                    └─────────┬───────────┘
                              │
                    ┌─────────┼─────────┐
                    │         │         │
                    ▼         ▼         ▼
        ┌─────────────┐ ┌───────────┐ ┌─────────────┐
        │     APNs    │ │    FCM    │ │ Web Push    │
        │   (iOS)     │ │ (Android) │ │   (Web)     │
        └─────────────┘ └───────────┘ └─────────────┘
```

## Components

### 1. API Gateway (`cmd/api-gateway/`)

**Responsibilities:**
- Accept HTTP requests for notifications
- Validate request payloads
- Manage user sessions
- Implement rate limiting
- Route messages to Kafka

**Key Features:**
- RESTful API endpoints
- JSON request/response handling
- Middleware for logging, CORS, recovery
- Graceful shutdown support
- Health check endpoint

**Endpoints:**
- `POST /api/v1/notifications` - Send notification
- `GET /api/v1/notifications/:id/status` - Check status
- `POST /api/v1/sessions` - Register user session
- `DELETE /api/v1/sessions/:user_id` - Unregister session
- `GET /health` - Health check

### 2. Consumer Service (`cmd/consumer/`)

**Responsibilities:**
- Process messages from Kafka queue
- Route notifications to appropriate platforms
- Handle retry logic and failure scenarios
- Update notification status in Redis

**Key Features:**
- Concurrent worker goroutines (10 workers by default)
- Platform-specific notification handlers
- Retry mechanism with exponential backoff
- Dead letter queue support (future enhancement)

### 3. Message Queue (Kafka - Simulated)

**Responsibilities:**
- Decouple API Gateway from Consumer
- Ensure message durability
- Enable horizontal scaling
- Handle high throughput (10,000+ notifications/minute)

**Implementation:**
- Currently simulated with Go channels for demo
- Ready for real Kafka integration
- Supports producer/consumer pattern
- Message persistence and replay capability

### 4. Cache Layer (Redis)

**Responsibilities:**
- Store user sessions and device tokens
- Implement rate limiting counters
- Cache notification statuses
- Store temporary data with TTL

**Key Features:**
- Session management with 24-hour TTL
- Rate limiting (100 notifications/minute per user)
- Notification status tracking
- Generic caching interface

### 5. Data Models (`pkg/models/`)

**Core Entities:**
- `NotificationRequest` - Incoming API requests
- `Notification` - Internal notification representation  
- `UserSession` - User device and platform information
- `APIResponse` - Standardized API responses

**Status Flow:**
```
Pending → Processing → Sent/Failed/Retry
```

## Scalability Design

### Horizontal Scaling

1. **API Gateway Scaling**
   - Multiple instances behind load balancer
   - Stateless design enables easy scaling
   - Shared Redis for session consistency

2. **Consumer Scaling**
   - Multiple consumer instances with different group IDs
   - Worker pool pattern within each instance
   - Kafka partitioning for parallel processing

3. **Infrastructure Scaling**
   - Redis clustering for cache scaling
   - Kafka partitioning for queue scaling
   - Container orchestration (Kubernetes ready)

### Performance Optimizations

1. **Message Batching**
   - Kafka producer batching
   - Consumer batch processing
   - Database bulk operations

2. **Connection Pooling**
   - Redis connection pooling
   - HTTP client reuse
   - Database connection management

3. **Caching Strategy**
   - User session caching
   - Configuration caching
   - Result caching with TTL

## Fault Tolerance

### Error Handling

1. **Retry Mechanism**
   - Exponential backoff
   - Maximum retry limits (3 attempts)
   - Dead letter queue for failed messages

2. **Circuit Breaker Pattern**
   - External service failure handling
   - Graceful degradation
   - Service health monitoring

3. **Graceful Shutdown**
   - Proper cleanup on termination
   - Message processing completion
   - Resource deallocation

### Data Consistency

1. **At-Least-Once Delivery**
   - Kafka message acknowledgment
   - Consumer offset management
   - Idempotent processing design

2. **Session Management**
   - TTL-based session expiration
   - Automatic cleanup
   - Consistent state across instances

## Security Considerations

### Current Implementation
- Input validation and sanitization
- CORS middleware
- Request ID tracking
- Structured logging

### Production Enhancements
- JWT-based authentication
- API key management
- Rate limiting per API key
- TLS/HTTPS encryption
- Request signing and verification

## Monitoring and Observability

### Logging
- Structured logging with logrus
- Configurable log levels
- Request/response logging
- Error tracking and correlation

### Metrics (Future)
- Notification throughput
- Processing latency
- Error rates
- Queue depths
- Consumer lag

### Health Checks
- API endpoint health
- Redis connectivity
- Kafka connectivity
- Consumer processing status

## Configuration Management

### Environment-based Config
- YAML configuration files
- Environment variable overrides
- Validation and defaults
- Hot reload capability (future)

### Deployment Configurations
- Development (local)
- Staging (docker-compose)
- Production (Kubernetes)

## Technology Stack

### Core Technologies
- **Go 1.21** - Primary programming language
- **Gin Framework** - HTTP web framework
- **Redis** - Caching and session storage
- **Kafka** - Message queuing (simulated)
- **Docker** - Containerization

### Supporting Libraries
- `logrus` - Structured logging
- `uuid` - Unique ID generation
- `yaml.v2` - Configuration parsing
- `redis/v8` - Redis client
- `gin-gonic/gin` - Web framework

## Future Enhancements

### Short Term
- Real Kafka integration
- WebSocket support for real-time updates
- Advanced retry mechanisms
- Metrics collection and dashboards

### Long Term
- Machine learning for optimal delivery timing
- A/B testing framework for notifications
- Advanced user segmentation
- Multi-region deployment support
- Message templates and localization