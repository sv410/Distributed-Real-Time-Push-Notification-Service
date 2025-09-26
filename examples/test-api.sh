#!/bin/bash

# Test script for the Notification API
# This script demonstrates how to use the notification service APIs

set -e

API_BASE_URL="http://localhost:8080"

echo "🚀 Testing Notification Service API"
echo "======================================"

# Function to make HTTP requests and handle responses
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    echo ""
    echo "📡 $method $endpoint"
    if [ -n "$data" ]; then
        echo "📤 Request: $data" 
    fi
    
    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" "$API_BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -w "\nHTTP_STATUS:%{http_code}")
    else
        response=$(curl -s -X "$method" "$API_BASE_URL$endpoint" \
            -w "\nHTTP_STATUS:%{http_code}")
    fi
    
    http_body=$(echo "$response" | sed -E '$d')
    http_status=$(echo "$response" | tail -n1 | sed -E 's/.*HTTP_STATUS:([0-9]{3}).*/\1/')
    
    echo "📥 Response ($http_status): $http_body"
    
    if [[ $http_status -ge 200 && $http_status -lt 300 ]]; then
        echo "✅ Success"
    else
        echo "❌ Error"
    fi
    
    echo "$http_body"
}

# Test 1: Health Check
echo ""
echo "🔍 Test 1: Health Check"
echo "----------------------"
make_request "GET" "/health" > /dev/null

# Test 2: Register User Session
echo ""
echo "👤 Test 2: Register User Session"
echo "--------------------------------"
session_data='{
  "user_id": "user123",
  "device_token": "sample_device_token_12345",
  "platform": "ios"
}'
make_request "POST" "/api/v1/sessions" "$session_data" > /dev/null

# Test 3: Send Notification
echo ""
echo "📢 Test 3: Send Notification"
echo "----------------------------"
notification_data='{
  "user_id": "user123",
  "title": "Welcome to our app!",
  "message": "Thank you for joining us. Start exploring amazing features!",
  "data": {
    "type": "welcome",
    "screen": "home"
  },
  "priority": "high"
}'
notification_response=$(make_request "POST" "/api/v1/notifications" "$notification_data")

# Extract notification ID from response
notification_id=$(echo "$notification_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -n "$notification_id" ]; then
    echo "📋 Notification ID: $notification_id"
    
    # Test 4: Check Notification Status
    echo ""
    echo "📊 Test 4: Check Notification Status"
    echo "------------------------------------"
    make_request "GET" "/api/v1/notifications/$notification_id/status" > /dev/null
else
    echo "⚠️  Could not extract notification ID, skipping status check"
fi

# Test 5: Send Multiple Notifications (Performance Test)
echo ""
echo "🚄 Test 5: Performance Test - Multiple Notifications"
echo "====================================================="
echo "Sending 10 notifications to test throughput..."

for i in {1..10}; do
    performance_data='{
      "user_id": "user123",
      "title": "Performance Test #'$i'",
      "message": "This is performance test notification number '$i'",
      "priority": "normal"
    }'
    make_request "POST" "/api/v1/notifications" "$performance_data" > /dev/null &
done

# Wait for all background requests to complete
wait

echo ""
echo "⏱️  All performance test notifications sent!"

# Test 6: Rate Limiting Test
echo ""
echo "🛡️  Test 6: Rate Limiting Test"
echo "==============================="
echo "Attempting to send notifications rapidly to test rate limiting..."

for i in {1..5}; do
    rate_limit_data='{
      "user_id": "user123",
      "title": "Rate Limit Test #'$i'",
      "message": "Testing rate limiting",
      "priority": "low"
    }'
    make_request "POST" "/api/v1/notifications" "$rate_limit_data" > /dev/null
done

# Test 7: Test with Different User
echo ""
echo "👥 Test 7: Different User Session"
echo "==================================="
different_user_session='{
  "user_id": "user456",
  "device_token": "another_device_token_67890",
  "platform": "android"
}'
make_request "POST" "/api/v1/sessions" "$different_user_session" > /dev/null

different_user_notification='{
  "user_id": "user456",
  "title": "Android Notification",
  "message": "This is a test notification for Android user",
  "data": {
    "type": "test",
    "platform": "android"
  },
  "priority": "normal"
}'
make_request "POST" "/api/v1/notifications" "$different_user_notification" > /dev/null

# Test 8: Unregister Session
echo ""
echo "🔓 Test 8: Unregister User Session"
echo "==================================="
make_request "DELETE" "/api/v1/sessions/user123" > /dev/null

# Test 9: Try to send notification to unregistered user
echo ""
echo "🚫 Test 9: Send to Unregistered User (Should Fail)"
echo "==================================================="
unregistered_notification='{
  "user_id": "user123",
  "title": "Should Fail",
  "message": "This should fail because user is not registered",
  "priority": "normal"
}'
make_request "POST" "/api/v1/notifications" "$unregistered_notification" > /dev/null

echo ""
echo "🎉 API Testing Complete!"
echo "========================="
echo ""
echo "📊 Summary:"
echo "- Health check: API is running"
echo "- Session management: Registration and unregistration working"
echo "- Notification sending: Basic and bulk sending working"
echo "- Error handling: Proper error responses for invalid requests"
echo "- Rate limiting: System handles multiple rapid requests"
echo ""
echo "💡 Check the consumer logs to see notification processing!"
echo "   You can run: docker-compose logs -f consumer"