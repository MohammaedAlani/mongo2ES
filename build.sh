#!/bin/bash

# Get the directory of the script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR

# Clean previous builds
rm -rf ./build

# Create build directory
mkdir -p ./build

# Copy config files
cp -r ./config ./build/

# Build the binary for Linux (most common server OS)
GOOS=linux GOARCH=amd64 go build -o ./build/mongo-to-elastic ./cmd/app

# Create logs directory in the build folder
mkdir -p ./build/logs

echo "Build completed successfully. Files are in the ./build directory."