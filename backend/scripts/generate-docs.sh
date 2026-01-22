#!/bin/bash

# Generate API Documentation Script
# This script generates OpenAPI documentation and validates it

set -e

echo "Generating API documentation..."

# Locate swag binary
if command -v swag >/dev/null 2>&1; then
    SWAG_BIN="swag"
elif [ -x "$(go env GOPATH)/bin/swag" ]; then
    SWAG_BIN="$(go env GOPATH)/bin/swag"
else
    echo "ERROR: swag not found. Install: go install github.com/swaggo/swag/cmd/swag@latest"
    exit 1
fi

# Generate Swagger documentation
echo "Generating Swagger/OpenAPI specification..."
$SWAG_BIN init -g cmd/api-service/main.go -o docs --parseDependency --parseInternal

# Validate the generated documentation
echo "✅ Validating generated documentation..."
if [ -f "docs/swagger.json" ]; then
    echo "✓ swagger.json generated successfully"
    
    # Check if the JSON is valid
    if jq empty docs/swagger.json 2>/dev/null; then
        echo "✓ swagger.json is valid JSON"
    else
        echo "❌ swagger.json is not valid JSON"
        exit 1
    fi
    
    # Check for required fields
    if jq -e '.info.title' docs/swagger.json > /dev/null; then
        echo "✓ API title is present"
    else
        echo "❌ API title is missing"
        exit 1
    fi
    
    if jq -e '.paths' docs/swagger.json > /dev/null; then
        echo "✓ API paths are present"
        
        # Count the number of documented endpoints
        path_count=$(jq '.paths | keys | length' docs/swagger.json)
        echo "📊 Documented endpoints: $path_count"
        
        if [ "$path_count" -lt 10 ]; then
            echo "⚠️  Warning: Only $path_count endpoints documented. Expected at least 10."
        fi
    else
        echo "❌ API paths are missing"
        exit 1
    fi
    
else
    echo "❌ swagger.json was not generated"
    exit 1
fi

if [ -f "docs/swagger.yaml" ]; then
    echo "✓ swagger.yaml generated successfully"
else
    echo "❌ swagger.yaml was not generated"
    exit 1
fi

# Generate SDK documentation
echo "📚 Generating SDK documentation..."
if [ -d "sdk/go" ]; then
    echo "✓ Go SDK structure exists"
    
    # Check if README exists
    if [ -f "sdk/go/README.md" ]; then
        echo "✓ Go SDK README exists"
    else
        echo "❌ Go SDK README is missing"
        exit 1
    fi
    
    # Check if client package exists
    if [ -f "sdk/go/client/client.go" ]; then
        echo "✓ Go SDK client package exists"
    else
        echo "❌ Go SDK client package is missing"
        exit 1
    fi
    
    # Check if examples exist
    if [ -d "sdk/go/examples" ]; then
        example_count=$(find sdk/go/examples -name "*.go" | wc -l)
        echo "✓ Go SDK examples exist ($example_count files)"
    else
        echo "❌ Go SDK examples are missing"
        exit 1
    fi
else
    echo "❌ Go SDK directory is missing"
    exit 1
fi

# Run documentation tests
echo "🧪 Running documentation tests..."
if go test ./docs -v; then
    echo "✓ Documentation tests passed"
else
    echo "❌ Documentation tests failed"
    exit 1
fi

echo ""
echo "🎉 Documentation generation completed successfully!"
echo ""
echo "📖 Access your API documentation at:"
echo "   • Swagger UI: http://localhost:8080/docs/"
echo "   • JSON spec: http://localhost:8080/docs/swagger.json"
echo "   • YAML spec: http://localhost:8080/docs/swagger.yaml"
echo ""
echo "📦 SDK Documentation:"
echo "   • Go SDK: sdk/go/README.md"
echo "   • Examples: sdk/go/examples/"
echo ""
echo "🔍 To serve the documentation locally:"
echo "   make run-api"
echo "   # Then visit http://localhost:8080/docs/"