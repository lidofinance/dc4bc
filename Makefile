test:
	@echo "Testing Go packages..."
	@go test ./... -cover

test-short:
	@echo "Testing Go packages..."
	@go test ./app/... -cover -short

mocks:
	@echo "Regenerate mocks..."
	@go generate ./...

build:
	@echo "Building dc4bc_d..."
	@go build -o dc4bc_d ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	@go build -o dc4bc_cli ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	@go build -o dc4bc_airgapped ./cmd/airgapped/main.go

.PHONY: mocks
