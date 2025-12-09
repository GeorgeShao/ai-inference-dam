#!/bin/bash

# Load test script - queues 10,000 requests to the chat completions endpoint

ENDPOINT="http://localhost:8080/v1/chat/completions"
TOTAL=10000

echo "Queuing $TOTAL requests to $ENDPOINT..."

START_TIME=$(date +%s.%N)

for i in $(seq 1 $TOTAL); do
  curl -s -X POST "$ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "gpt-4",
      "messages": [{"role": "user", "content": "Hello! Request '"$i"'"}]
    }' > /dev/null

  # Print progress every 100 requests
  if (( i % 100 == 0 )); then
    echo "Queued $i / $TOTAL requests"
  fi
done

END_TIME=$(date +%s.%N)
RUNTIME=$(echo "$END_TIME - $START_TIME" | bc)
RPS=$(echo "scale=2; $TOTAL / $RUNTIME" | bc)

echo ""
echo "=== Benchmark Stats ==="
echo "Total requests: $TOTAL"
echo "Runtime: ${RUNTIME}s"
echo "Requests/sec: $RPS"