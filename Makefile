# Get all repository names under an organization and store in a yaml file
get-org-repos:
	@if [ -z "$(ORG)" ] || [ -z "$(TOKEN)" ]; then \
		echo "Usage: make get-org-repos ORG=my-org TOKEN=ghp_xxx [OUTPUT=repos.yaml]"; \
		exit 1; \
	fi
	docker-compose run --rm --entrypoint /app/get_org_repos organization-checker \
		-token $(TOKEN) -org $(ORG) -output $${OUTPUT:-/workspace/repos.yaml}
.PHONY: build run shell clean init help organization-check

# Create org code security configuration from yaml
create-org-config:
	@if [ -z "$(ORG)" ] || [ -z "$(TOKEN)" ] || [ -z "$(YAML)" ]; then \
		echo "Usage: make create-org-config ORG=my-org TOKEN=ghp_xxx YAML=/workspace/sample_org_config.yaml"; \
		exit 1; \
	fi
	docker-compose run --rm --entrypoint /app/create_org_config organization-checker \
		-org $(ORG) -token $(TOKEN) -yaml $(YAML)
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
	@echo "  get-org-repos  - Get all repository names under an organization and store in a yaml file"
	@echo "  create-org-config - Create org code security configuration from a yaml file"
	@echo "  shell          - Open a shell in the container"
	@echo "  clean          - Clean up Docker resources"
	@echo ""
	@echo "Note: Set GITHUB_TOKEN environment variable first"
	@echo ""
	@echo "Quick start for testing:"
	@echo "  export GITHUB_TOKEN=your_token_here"
	@echo "  make organization-check"
