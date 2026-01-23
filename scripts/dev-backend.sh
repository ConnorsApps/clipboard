#!/bin/bash

set -e

if ! which wgo &> /dev/null; then
    go install github.com/bokwoon95/wgo@latest
fi

# Cleanup function to kill child processes on exit
cleanup() {
    echo ""
    echo "Shutting down..."
    kill $BACKEND_PID 2>/dev/null
    wait
    exit 0
}

trap cleanup SIGINT SIGTERM


# Start backend with wgo (live reload on .go file changes)
echo "Starting backend with live reload..."
wgo run .