#!/bin/bash

hey -n 10000 -c 100 -m POST -H "Content-Type: application/json" \
-d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}' \
http://localhost:8080/v1/chat/completions