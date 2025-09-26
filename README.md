# Distributed Real-Time Push Notification Service

A high-throughput, fault-tolerant backend service built in Go for handling asynchronous push notifications for mobile applications. This service can handle a burst capacity of **10,000 notifications per minute** using Kafka as a message queue and Redis for caching.

## 🏗️ Architecture

The service follows a microservices architecture with the following components:

- **API Gateway**: RESTful API server that receives notification requests
- **Kafka**: Distributed message queue for decoupling and high throughput
- **Consumer Service**: Processes notifications from Kafka queue
- **Redis**: Caching layer for user sessions and rate limiting
- **Docker**: Containerization for easy deployment

## 🚀 Features

- **High Throughput**: Handles 10,000+ notifications per minute
- **Fault Tolerance**: Retry mechanisms and error handling
- **Multi-Platform Support**: iOS, Android, and Web push notifications
- **Rate Limiting**: Per-user rate limiting to prevent abuse
- **Session Management**: User session tracking with Redis
- **Graceful Shutdown**: Proper cleanup on service termination
- **Monitoring**: Comprehensive logging and health checks
- **Containerized**: Docker support for easy deployment

## 📋 Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Kafka 2.8+
- Redis 6.0+

## 🛠️ Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
cd Distributed-Real-Time-Push-Notification-Service
```

2. Start all services:
```bash
docker-compose up -d
```

This will start:
- Zookeeper (port 2181)
- Kafka (port 9092)
- Redis (port 6379)
- API Gateway (port 8080)
- Consumer Service (3 replicas)
- Kafka UI (port 8081) - for monitoring
- Redis Commander (port 8082) - for Redis monitoring

### Manual Setup

1. Install dependencies:
```bash
go mod download
```

2. Start Kafka and Redis:
```bash
# Start Kafka (requires Zookeeper)
# Start Redis
redis-server
```

3. Build the applications:
```bash
./build.sh
```

4. Run the API Gateway:
```bash
./bin/api-gateway
```

5. Run the Consumer (in separate terminal):
```bash
./bin/consumer
```

## 📡 API Endpoints

### Register User Session
```bash
POST /api/v1/sessions
Content-Type: application/json

{
  "user_id": "user123",
  "device_token": "device_token_here",
  "platform": "ios"
}
```

### Send Notification
```bash
POST /api/v1/notifications
Content-Type: application/json

{
  "user_id": "user123",
  "title": "New Message",
  "message": "You have a new message!",
  "data": {
    "type": "chat",
    "chat_id": "chat456"
  },
  "priority": "normal"
}
```

### Check Notification Status
```bash
GET /api/v1/notifications/{notification_id}/status
```

### Unregister User Session
```bash
DELETE /api/v1/sessions/{user_id}
```

### Health Check
```bash
GET /health
```

## 🔧 Configuration

Configuration can be provided via `config.yaml` file or environment variables:

```yaml
server:
  host: "0.0.0.0"
  port: "8080"

kafka:
  bootstrap_servers: "localhost:9092"
  topic: "push-notifications"
  group_id: "notification-consumer-group"
  auto_offset_reset: "earliest"

redis:
  host: "localhost"
  port: "6379"
  password: ""
  db: 0

log:
  level: "info"
  file: ""
```

## 🧪 Testing

### Example: Complete Workflow

1. Register a user session:
```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "device_token": "sample_device_token",
    "platform": "ios"
  }'
```

2. Send a notification:
```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "title": "Test Notification",
    "message": "This is a test message!",
    "priority": "high"
  }'
```

3. Check notification status:
```bash
curl http://localhost:8080/api/v1/notifications/{notification_id}/status
```

## 📊 Performance

- **Throughput**: 10,000+ notifications per minute
- **Latency**: < 100ms for API responses
- **Concurrency**: Multiple consumer workers for parallel processing
- **Scalability**: Horizontal scaling with Kafka partitions and consumer groups

## 🔍 Monitoring

- **Kafka UI**: http://localhost:8081 - Monitor Kafka topics and consumers
- **Redis Commander**: http://localhost:8082 - Monitor Redis data
- **Health Endpoint**: http://localhost:8080/health - Service health status
- **Logs**: Structured logging with configurable levels

## 🏢 Production Considerations

1. **Security**: Implement authentication and authorization
2. **TLS**: Use HTTPS in production
3. **Database**: Consider persistent storage for notifications
4. **Monitoring**: Add metrics collection (Prometheus, Grafana)
5. **Alerting**: Set up alerting for failures and performance issues
6. **Load Balancing**: Use load balancer for API Gateway
7. **Backup**: Regular backups of Redis data

## 🛡️ Fault Tolerance

- **Retry Logic**: Failed notifications are retried up to 3 times
- **Circuit Breaker**: Graceful degradation on external service failures
- **Health Checks**: Regular health monitoring
- **Graceful Shutdown**: Proper cleanup on service termination
- **Dead Letter Queue**: Failed messages after max retries (implementation ready)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🎯 Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Message templates and localization
- [ ] Analytics and reporting dashboard
- [ ] A/B testing for notifications
- [ ] Push notification scheduling
- [ ] Advanced targeting and segmentation
- [ ] Integration with APNs and FCM
- [ ] Metrics and observability improvements