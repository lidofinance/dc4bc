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

build-darwin:
	@echo "Building dc4bc_d..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_d_darwin ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_cli_darwin ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_airgapped_darwin ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_prysm_compatibility_checker_darwin ./cmd/prysm_compatibility_checker/
	@echo "Building Airgapped state cleaner..."
	GOOS=darwin GOARCH=amd64 go build -o airgapped_state_cleaner_darwin ./cmd/airgapped_state_cleaner/

build-linux:
	@echo "Building dc4bc_d..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_d_linux ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_cli_linux ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_airgapped_linux ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_prysm_compatibility_checker_linux ./cmd/prysm_compatibility_checker/
	@echo "Building Airgapped state cleaner..."
	GOOS=linux GOARCH=amd64 go build -o airgapped_state_cleaner_linux ./cmd/airgapped_state_cleaner/

build:
	@echo "Building dc4bc_d..."
	go build -o dc4bc_d ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	go build -o dc4bc_cli ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	go build -o dc4bc_airgapped ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	go build -o dc4bc_prysm_compatibility_checker_linux ./cmd/prysm_compatibility_checker/
	@echo "Building Airgapped state cleaner..."
	go build -o airgapped_state_cleaner ./cmd/airgapped_state_cleaner/

.PHONY: mocks
