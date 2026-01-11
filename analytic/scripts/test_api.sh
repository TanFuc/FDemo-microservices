#!/bin/bash

# Test script for Analytics Service API endpoints

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "Testing Analytics Service API"
echo "=============================="
echo ""

# Health check
echo "1. Health Check"
curl -s "$BASE_URL/health" | jq .
echo ""

# Collect event - view_item
echo "2. Collect Event (view_item)"
curl -s -X POST "$BASE_URL/analytics/collect" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "event_type": "view_item",
    "metadata": "{\"sku_id\": \"SKU-001\", \"category\": \"electronics\"}",
    "url": "https://example.com/products/SKU-001"
  }' | jq .
echo ""

# Collect event - add_to_cart
echo "3. Collect Event (add_to_cart)"
curl -s -X POST "$BASE_URL/analytics/collect" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "event_type": "add_to_cart",
    "metadata": "{\"sku_id\": \"SKU-001\", \"quantity\": 2}",
    "url": "https://example.com/cart"
  }' | jq .
echo ""

# Collect event - checkout_start
echo "4. Collect Event (checkout_start)"
curl -s -X POST "$BASE_URL/analytics/collect" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "event_type": "checkout_start",
    "metadata": "{\"cart_total\": 199.99}",
    "url": "https://example.com/checkout"
  }' | jq .
echo ""

# Wait for events to be flushed
echo "Waiting 6 seconds for events to flush to ClickHouse..."
sleep 6

# Get product views
echo "5. Get Product Views (SKU-001)"
curl -s "$BASE_URL/analytics/products/SKU-001/views" | jq .
echo ""

# Get conversion rate
echo "6. Get Conversion Rate"
curl -s "$BASE_URL/analytics/conversion" | jq .
echo ""

echo "=============================="
echo "API Tests Complete"
