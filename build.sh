#!/bin/bash

# Build script for Distributed Counter System (Unix/Linux)

echo "Building Distributed Counter System..."
go build -o distributed-counter

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful! Binary created: distributed-counter"
chmod +x distributed-counter
