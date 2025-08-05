.PHONY: build run shell clean init help organization-check

# Initialize go.sum file
init:
	docker run --rm -v $(PWD):/workspace -w /workspace golang:1.21-alpine sh -c "apk add --no-cache git && go mod tidy"

# Build the Docker image
build: init
	docker-compose build

# Test enterprise and organization access
organization-check:
	docker-compose run --rm --entrypoint ./organization-check organization-checker

# Open a shell in the container for development
shell:
	docker-compose run --rm organization-checker sh

# Clean up Docker resources
clean:
	docker-compose down --rmi all --volumes --remove-orphans

# Help command
help:
	@echo "Available commands:"
	@echo "  init           - Initialize go.sum file"
	@echo "  build          - Build the Docker image"
	@echo "  run            - Run the secret scanning tool"
	@echo "  organization-check - Test enterprise and organization access"
	@echo "  shell          - Open a shell in the container"
	@echo "  clean          - Clean up Docker resources"
	@echo ""
	@echo "Note: Set GITHUB_TOKEN environment variable first"
	@echo ""
	@echo "Quick start for testing:"
	@echo "  export GITHUB_TOKEN=your_token_here"
	@echo "  make organization-check"
