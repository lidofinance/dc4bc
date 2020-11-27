.ONESHELL:

test:
	@echo "Testing Go packages..."
	@go test ./... -cover

test-short:
	@echo "Testing Go packages..."
	@go test ./... -cover -short

mocks:
	@echo "Regenerate mocks..."
	@go generate ./...

build-node:
	@echo "Building dc4bc_d..."
	@go build -o dc4bc_d ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	@go build -o dc4bc_cli ./cmd/dc4bc_cli/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	@go build -o dc4bc_prysm_compatibility_checker ./cmd/prysm_compatibility_checker/

build-airgapped-machine:
	@echo "Building dc4bc_airgapped..."
	@go build -o dc4bc_airgapped ./cmd/airgapped/

build-local:
	@echo "Building dc4bc_d..."
	go build -o dc4bc_d_darwin ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	go build -o dc4bc_cli_darwin ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	go build -o dc4bc_airgapped_darwin ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	go build -o dc4bc_prysm_compatibility_checker_darwin ./cmd/prysm_compatibility_checker/

run-client-node:
	@docker rm -f client_node > /dev/null 2>&1 || true
	@docker build . --tag client_node -f node.Dockerfile
	@docker run -it -e STORAGE_TOPIC=foo -e STORAGE_DBDSN=$(STORAGE_DBDSN) -e USERNAME=$(USERNAME) --name client_node -v $(DATA_DIR):/go/src/shared -p $(QR_READER_PORT):9090 client_node:latest

run-client-node-bash:
	@docker exec -it client_node bash

run-airgapped-machine:
	@docker rm -f airgapped_machine > /dev/null 2>&1 || true
	@docker build . --tag airgapped_machine -f airgapped_machine.Dockerfile
	@docker run -it -e PASSWORD_EXPIRATION=$(PASSWORD_EXPIRATION) -e USERNAME=$(USERNAME) --name airgapped_machine -v $(DATA_DIR):/go/src/shared airgapped_machine:latest

.PHONY: mocks
