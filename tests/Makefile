# Variables
.PHONY: all test-curl

# Default target
all: test-curl

# Variables
PORT := 9090
DEST_FILENAME := "chart.zip"


test-curl:
	@curl -X POST http://localhost:$(PORT)/generate \
  		-H "Content-Type: application/json" \
  		--data-binary @chartpress.json -o $(DEST_FILENAME)
