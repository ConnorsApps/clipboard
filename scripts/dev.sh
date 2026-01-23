#!/bin/bash

# Development script - runs backend (with wgo live reload) and frontend concurrently

set -e

if ! which wgo &> /dev/null; then
    go install github.com/bokwoon95/wgo@latest
fi

# Cleanup function to kill child processes on exit
cleanup() {
    echo ""
    echo "Shutting down..."
    kill $FRONTEND_PID 2>/dev/null
    kill $BACKEND_PID 2>/dev/null
    wait
    exit 0
}

trap cleanup SIGINT SIGTERM

# Start frontend dev server
echo "Starting frontend..."
cd frontend && npm run dev &
FRONTEND_PID=$!

# Start backend with wgo (live reload on .go file changes)
echo "Starting backend with live reload..."
wgo run ./cmd/server &
BACKEND_PID=$!

# Wait for both processes
wait
