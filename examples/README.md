# Examples and Usage

This directory contains examples and scripts to help you understand and test the notification service.

## Quick Start

### 1. Start the Service

Using Docker Compose (Recommended):
```bash
docker-compose up -d
```

Or manually (requires local Redis and Kafka):
```bash
# Terminal 1: Start API Gateway
make run-api

# Terminal 2: Start Consumer
make run-consumer
```

### 2. Test the API

Run the comprehensive test script:
```bash
./examples/test-api.sh
```

### 3. Manual API Testing

#### Register a User Session
```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "device_token": "sample_device_token",
    "platform": "ios"
  }'
```

#### Send a Notification
```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "title": "Hello World!",
    "message": "This is a test notification",
    "priority": "high",
    "data": {
      "type": "test",
      "action": "open_app"
    }
  }'
```

#### Check Notification Status
```bash
curl http://localhost:8080/api/v1/notifications/{notification_id}/status
```

#### Health Check
```bash
curl http://localhost:8080/health
```

## Performance Testing

The service is designed to handle **10,000 notifications per minute**. Here's how to test high throughput:

### Bulk Send Test
```bash
# Send 100 notifications concurrently
for i in {1..100}; do
  curl -X POST http://localhost:8080/api/v1/notifications \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"user123\",
      \"title\": \"Bulk Test #$i\",
      \"message\": \"Performance test notification\",
      \"priority\": \"normal\"
    }" &
done
wait
```

### Load Testing with Apache Bench
```bash
# Install Apache Bench
sudo apt-get install apache2-utils

# Create test payload
echo '{
  "user_id": "user123",
  "title": "Load Test",
  "message": "Load testing notification",
  "priority": "normal"
}' > notification.json

# Run load test (1000 requests, 50 concurrent)
ab -n 1000 -c 50 -p notification.json -T application/json \
   http://localhost:8080/api/v1/notifications
```

## Monitoring

### View Logs
```bash
# Docker Compose logs
docker-compose logs -f api-gateway
docker-compose logs -f consumer

# Or all services
docker-compose logs -f
```

### Monitor Queues and Cache
- **Kafka UI**: http://localhost:8081
- **Redis Commander**: http://localhost:8082

### Service Health
```bash
curl http://localhost:8080/health
```

## Advanced Usage

### Scheduled Notifications
```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "title": "Scheduled Notification",
    "message": "This will be sent later",
    "schedule_at": "2024-01-01T12:00:00Z",
    "priority": "normal"
  }'
```

### Different Platform Notifications
```bash
# iOS
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id": "ios_user", "device_token": "ios_token", "platform": "ios"}'

# Android
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id": "android_user", "device_token": "fcm_token", "platform": "android"}'

# Web
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id": "web_user", "device_token": "web_push_token", "platform": "web"}'
```

### Priority Notifications
```bash
# High Priority (processed first)
curl -X POST http://localhost:8080/api/v1/notifications \
  -d '{"user_id": "user123", "title": "Urgent!", "message": "High priority message", "priority": "high"}'

# Normal Priority
curl -X POST http://localhost:8080/api/v1/notifications \
  -d '{"user_id": "user123", "title": "Info", "message": "Normal message", "priority": "normal"}'

# Low Priority
curl -X POST http://localhost:8080/api/v1/notifications \
  -d '{"user_id": "user123", "title": "FYI", "message": "Low priority message", "priority": "low"}'
```

## Troubleshooting

### Common Issues

1. **Connection Refused Errors**
   - Ensure Redis and Kafka are running
   - Check Docker container status: `docker-compose ps`

2. **User Session Not Found**
   - Register the user session first before sending notifications
   - Check if session expired (24-hour TTL)

3. **Rate Limit Exceeded**
   - Wait a minute before sending more notifications
   - Default limit: 100 notifications per minute per user

4. **High Memory Usage**
   - Check queue sizes in Kafka UI
   - Ensure consumers are processing messages

### Debug Mode
Set log level to debug in config.yaml:
```yaml
log:
  level: "debug"
```

### Reset Everything
```bash
docker-compose down -v  # Removes volumes too
docker-compose up -d
```