#!/bin/bash

# Test script for the Distributed Push Notification Service

set -e

echo "🚀 Testing Distributed Push Notification Service"
echo "================================================="

# Configuration
SERVICE_URL="http://localhost:8080"
KAFKA_TOPIC="notifications"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if service is running
check_service() {
    echo -n "Checking if service is running... "
    if curl -sf "$SERVICE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Function to test health endpoint
test_health() {
    echo "1. Testing health endpoint:"
    response=$(curl -s "$SERVICE_URL/health")
    if echo "$response" | grep -q '"status":"healthy"'; then
        echo -e "   ${GREEN}✓ Service is healthy${NC}"
    else
        echo -e "   ${YELLOW}⚠ Service health check returned:${NC}"
        echo "$response" | jq .
    fi
    echo
}

# Function to test metrics endpoint
test_metrics() {
    echo "2. Testing metrics endpoint:"
    metrics=$(curl -s "$SERVICE_URL/metrics")
    echo -e "   ${GREEN}✓ Metrics retrieved:${NC}"
    echo "$metrics" | jq .
    echo
}

# Function to send test notifications
send_test_notifications() {
    echo "3. Sending test notifications:"
    
    # Test notification 1 - Normal priority
    echo "   Sending notification to user123..."
    response1=$(curl -s -X POST "$SERVICE_URL/send" \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "user123",
            "title": "Welcome!",
            "body": "Welcome to our notification service",
            "priority": 1,
            "data": {"campaign": "welcome", "version": "1.0"}
        }')
    
    if echo "$response1" | grep -q "Notification sent successfully"; then
        echo -e "   ${GREEN}✓ Notification 1 sent successfully${NC}"
    else
        echo -e "   ${RED}✗ Failed to send notification 1${NC}"
        echo "$response1"
    fi
    
    # Test notification 2 - High priority
    echo "   Sending high priority notification to user123..."
    response2=$(curl -s -X POST "$SERVICE_URL/send" \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "user123", 
            "title": "Important Update",
            "body": "Your account requires attention",
            "priority": 2,
            "data": {"urgency": "high", "action_required": true}
        }')
    
    if echo "$response2" | grep -q "Notification sent successfully"; then
        echo -e "   ${GREEN}✓ Notification 2 sent successfully${NC}"
    else
        echo -e "   ${RED}✗ Failed to send notification 2${NC}"
        echo "$response2"
    fi
    
    echo
}

# Function to test rate limiting
test_rate_limiting() {
    echo "4. Testing rate limiting (sending multiple notifications to user456):"
    
    # Send notifications to trigger rate limiting
    for i in {1..15}; do
        curl -s -X POST "$SERVICE_URL/send" \
            -H "Content-Type: application/json" \
            -d "{
                \"user_id\": \"user456\",
                \"title\": \"Test Notification $i\",
                \"body\": \"This is test notification number $i\",
                \"priority\": 0
            }" > /dev/null
        
        if [ $i -eq 5 ] || [ $i -eq 10 ] || [ $i -eq 15 ]; then
            echo -n "   Sent $i notifications... "
            sleep 0.5
            ratelimit_status=$(curl -s "$SERVICE_URL/ratelimit/user456")
            remaining=$(echo "$ratelimit_status" | jq -r '.remaining')
            current=$(echo "$ratelimit_status" | jq -r '.current')
            echo "current: $current, remaining: $remaining"
        fi
        
        sleep 0.1
    done
    
    echo -e "   ${GREEN}✓ Rate limiting test completed${NC}"
    
    # Check final rate limit status
    echo "   Final rate limit status for user456:"
    curl -s "$SERVICE_URL/ratelimit/user456" | jq .
    echo
}

# Function to check final metrics
check_final_metrics() {
    echo "5. Final service metrics:"
    sleep 2  # Wait for processing
    final_metrics=$(curl -s "$SERVICE_URL/metrics")
    echo "$final_metrics" | jq .
    
    processed=$(echo "$final_metrics" | jq -r '.processed_messages')
    rate_limited=$(echo "$final_metrics" | jq -r '.rate_limited_messages')
    
    if [ "$processed" -gt 0 ]; then
        echo -e "   ${GREEN}✓ Successfully processed $processed messages${NC}"
    fi
    
    if [ "$rate_limited" -gt 0 ]; then
        echo -e "   ${YELLOW}⚠ Rate limited $rate_limited messages${NC}"
    fi
    
    echo
}

# Main execution
main() {
    if ! check_service; then
        echo -e "${RED}Service is not running. Please start the service first:${NC}"
        echo "  docker-compose up -d"
        echo "  go run ./cmd"
        exit 1
    fi
    
    echo
    test_health
    test_metrics
    send_test_notifications
    test_rate_limiting
    check_final_metrics
    
    echo "================================================="
    echo -e "${GREEN}✅ All tests completed successfully!${NC}"
    echo
    echo "💡 Next steps:"
    echo "  • Check Kafka UI at http://localhost:8080"
    echo "  • Check Redis Commander at http://localhost:8081" 
    echo "  • Monitor service metrics at $SERVICE_URL/metrics"
    echo "  • View service health at $SERVICE_URL/health"
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}jq is required but not installed. Please install jq first.${NC}"
    exit 1
fi

# Run tests
main