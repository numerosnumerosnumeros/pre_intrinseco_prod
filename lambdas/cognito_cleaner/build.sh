#!/bin/bash

# Call from root directory: ./lambdas/cognito_cleaner/build.sh

ROOT_DIR=$(pwd)
cd "$ROOT_DIR/lambdas/cognito_cleaner"
BUILD_DIR="bin"
FUNCTION_NAME="cognito_cleaner"

echo "🧹 Cleaning build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo "🔨 Building for AWS Lambda..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o $BUILD_DIR/bootstrap

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

cd $BUILD_DIR && zip cognito_cleaner.zip bootstrap && cd ..

if [ $? -ne 0 ]; then
    echo "❌ Zip creation failed!"
    exit 1
fi

echo "✅ Lambda build complete. Output: $BUILD_DIR/cognito_cleaner.zip"

echo "🚀 Uploading to Lambda function: $FUNCTION_NAME"
aws lambda update-function-code \
    --function-name "$FUNCTION_NAME" \
    --zip-file "fileb://$BUILD_DIR/cognito_cleaner.zip"

if [ $? -eq 0 ]; then
    echo "✅ Lambda function updated successfully!"
else
    echo "❌ Failed to update Lambda function!"
    exit 1
fi
