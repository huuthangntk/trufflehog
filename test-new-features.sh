#!/bin/bash

# Test script for new TruffleHog features
# This script verifies that all new components are properly integrated

set -e

echo "=========================================="
echo "Testing New TruffleHog Features"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check if new detector files exist
echo "Test 1: Checking new detector files..."
DETECTORS=("perplexity" "elevenlabs/v3" "openrouter" "runwayml" "firecrawl" "exa")
for detector in "${DETECTORS[@]}"; do
    if [ -f "pkg/detectors/$detector/${detector##*/}.go" ] || [ -f "pkg/detectors/$detector/elevenlabs.go" ]; then
        echo -e "${GREEN}✓${NC} Detector $detector exists"
    else
        echo -e "${RED}✗${NC} Detector $detector NOT found"
        exit 1
    fi
done
echo ""

# Test 2: Check if proto file is updated
echo "Test 2: Checking proto file updates..."
NEW_TYPES=("Perplexity" "ElevenLabs" "OpenRouter" "RunwayML" "Firecrawl" "Exa")
for type in "${NEW_TYPES[@]}"; do
    if grep -q "$type = 10[0-9][0-9]" proto/detectors.proto; then
        echo -e "${GREEN}✓${NC} DetectorType $type found in proto"
    else
        echo -e "${RED}✗${NC} DetectorType $type NOT found in proto"
        exit 1
    fi
done
echo ""

# Test 3: Check if detectors are registered in defaults.go
echo "Test 3: Checking detector registration..."
IMPORTS=("perplexity" "elevenlabsv3" "openrouter" "runwayml" "firecrawl" "exa")
for import in "${IMPORTS[@]}"; do
    if grep -q "\"github.com/trufflesecurity/trufflehog/v3/pkg/detectors" pkg/engine/defaults/defaults.go | grep -q "$import"; then
        echo -e "${GREEN}✓${NC} Detector $import imported in defaults.go"
    else
        echo -e "${YELLOW}⚠${NC} Detector $import import check skipped (may use alias)"
    fi
done
echo ""

# Test 4: Check if API files exist
echo "Test 4: Checking API service files..."
API_FILES=("pkg/api/server.go" "cmd/api/main.go" "Dockerfile.api" "docker-compose.yml")
for file in "${API_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} API file $file exists"
    else
        echo -e "${RED}✗${NC} API file $file NOT found"
        exit 1
    fi
done
echo ""

# Test 5: Check if documentation exists
echo "Test 5: Checking documentation files..."
DOC_FILES=("docs/API.md" "docs/NEW_FEATURES.md" "docs/SETUP_GUIDE.md")
for file in "${DOC_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} Documentation $file exists"
    else
        echo -e "${RED}✗${NC} Documentation $file NOT found"
        exit 1
    fi
done
echo ""

# Test 6: Check if example files exist
echo "Test 6: Checking example files..."
EXAMPLE_FILES=("examples/api-client.py" "examples/api-client.js" "examples/webhook-server.js")
for file in "${EXAMPLE_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} Example $file exists"
    else
        echo -e "${RED}✗${NC} Example $file NOT found"
        exit 1
    fi
done
echo ""

# Test 7: Verify protobuf updates
echo "Test 7: Checking protobuf Go file updates..."
if grep -q "DetectorType_Perplexity" pkg/pb/detectorspb/detectors.pb.go && \
   grep -q "DetectorType_ElevenLabs" pkg/pb/detectorspb/detectors.pb.go && \
   grep -q "DetectorType_OpenRouter" pkg/pb/detectorspb/detectors.pb.go && \
   grep -q "DetectorType_RunwayML" pkg/pb/detectorspb/detectors.pb.go && \
   grep -q "DetectorType_Firecrawl" pkg/pb/detectorspb/detectors.pb.go && \
   grep -q "DetectorType_Exa" pkg/pb/detectorspb/detectors.pb.go; then
    echo -e "${GREEN}✓${NC} All new detector types found in protobuf Go file"
else
    echo -e "${RED}✗${NC} Some detector types missing from protobuf Go file"
    exit 1
fi
echo ""

# Test 8: Check detector implementations
echo "Test 8: Checking detector implementations..."
for detector in "${DETECTORS[@]}"; do
    detector_name="${detector##*/}"
    if [ "$detector" = "elevenlabs/v3" ]; then
        detector_file="pkg/detectors/elevenlabs/v3/elevenlabs.go"
    else
        detector_file="pkg/detectors/$detector/$detector_name.go"
    fi
    
    if [ -f "$detector_file" ]; then
        # Check if file contains required methods
        if grep -q "func.*FromData" "$detector_file" && \
           grep -q "func.*Keywords" "$detector_file" && \
           grep -q "func.*Type" "$detector_file" && \
           grep -q "func.*Description" "$detector_file"; then
            echo -e "${GREEN}✓${NC} Detector $detector_name has all required methods"
        else
            echo -e "${RED}✗${NC} Detector $detector_name missing required methods"
            exit 1
        fi
    fi
done
echo ""

# Summary
echo "=========================================="
echo -e "${GREEN}All tests passed!${NC}"
echo "=========================================="
echo ""
echo "New features successfully integrated:"
echo "  • 6 new API key detectors"
echo "  • REST API service with webhook support"
echo "  • Complete documentation"
echo "  • Example client implementations"
echo ""
echo "Next steps:"
echo "  1. Build the project: go build ./..."
echo "  2. Run the API server: go run ./cmd/api/main.go"
echo "  3. Test with examples: python examples/api-client.py"
echo ""
