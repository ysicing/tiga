#!/bin/bash

# Swagger API Documentation Generator
# This script generates Swagger/OpenAPI documentation from code comments

set -e

echo "🔧 Installing swag if not already installed..."
if ! command -v swag &> /dev/null; then
    echo "Installing swag..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

echo "📝 Generating Swagger documentation..."
swag init \
    --dir ./cmd/tiga,./internal/api/handlers,./internal/api/handlers/instances,./internal/api/handlers/minio,./internal/api/handlers/database,./internal/api/handlers/docker \
    --generalInfo main.go \
    --output ./docs/swagger \
    --parseDependency \
    --parseInternal

echo "✅ Swagger documentation generated successfully!"
echo "📄 Files generated:"
echo "   - docs/swagger/swagger.json"
echo "   - docs/swagger/swagger.yaml"
echo "   - docs/swagger/docs.go"
echo ""
echo "🌐 To view the documentation:"
echo "   1. Start the server: go run cmd/tiga/main.go"
echo "   2. Open browser: http://localhost:12306/swagger/index.html"
