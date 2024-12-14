#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Application Name
APP_NAME="Crop-Disease-Predictor"

# Output Directory
BUILD_DIR="build"

# Check for Go installation
if ! command -v go &> /dev/null
then
    echo "Go is not installed. Please install Go to proceed."
    exit 1
fi

echo "Go is installed. Proceeding with build."

# Install fyne if not already installed
if ! command -v fyne &> /dev/null
then
    echo "Installing fyne CLI..."
    go install fyne.io/fyne/v2/cmd/fyne@latest
fi

echo "fyne CLI is installed."

# Create build directory if it doesn't exist
if [ ! -d "$BUILD_DIR" ]; then
    mkdir $BUILD_DIR
fi

# Build the application
echo "Building the application..."
go mod tidy # Ensure all dependencies are fetched
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME main.go

echo "Build successful. Executable is located at $BUILD_DIR/$APP_NAME"

# Copy assets to build directory
if [ -d "assets" ]; then
    cp -r assets $BUILD_DIR/
    echo "Assets copied to build directory."
else
    echo "No assets directory found. Skipping asset copy."
fi

# Package resources using fyne CLI
if [ -f "assets/icon.jpg" ]; then
    echo "Packaging resources with fyne package..."
    fyne package -os linux -name "$APP_NAME" -icon "assets/icon.jpg" -exe "$BUILD_DIR/$APP_NAME"
    echo "Packaging complete."
else
    echo "No icon found at assets/icon.png. Packaging skipped."
fi

echo "Build script completed."
