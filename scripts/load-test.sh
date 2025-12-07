#!/bin/bash

# Load test script - queues 10,000 requests to the chat completions endpoint

ENDPOINT="http://localhost:8080/v1/chat/completions"
TOTAL=10000

echo "Queuing $TOTAL requests to $ENDPOINT..."

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

echo "Done! Queued $TOTAL requests."