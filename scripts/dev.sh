#!/bin/bash

set -e

if ! which wgo &> /dev/null; then
    go install github.com/bokwoon95/wgo@latest
fi

cleanup() {
    echo ""
    echo "Shutting down..."
    kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
    wait
    exit 0
}

trap cleanup SIGINT SIGTERM

echo "Starting backend with live reload..."
wgo run . &
BACKEND_PID=$!

echo "Starting frontend..."
(cd frontend && npm run dev) &
FRONTEND_PID=$!

wait
