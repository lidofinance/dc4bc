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
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dc4bc_airgapped_darwin ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_prysm_compatibility_checker_darwin ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_dkg_reinitializer_darwin ./cmd/dkg_reinitializer/

build-linux:
	@echo "Building dc4bc_d..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_d_linux ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_cli_linux ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dc4bc_airgapped_linux ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_prysm_compatibility_checker_linux ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_dkg_reinitializer_linux ./cmd/dkg_reinitializer/

build:
	@echo "Building dc4bc_d..."
	go build -o dc4bc_d ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	go build -o dc4bc_cli ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	CGO_ENABLED=0 go build -o dc4bc_airgapped ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	go build -o dc4bc_prysm_compatibility_checker ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	go build -o dc4bc_dkg_reinitializer ./cmd/dkg_reinitializer/

.PHONY: mocks