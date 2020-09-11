test:
	@echo "Testing Go packages..."
	@go test ./... -cover

test-short:
	@echo "Testing Go packages..."
	@go test ./app/... -cover -short

mocks:
	@echo "Regenerate mocks..."
	@go generate ./...

build-darwin:
	@echo "Building dc4bc_d..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_d_darwin ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_cli_darwin ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_airgapped_darwin ./cmd/airgapped/main.go

build-linux:
	@echo "Building dc4bc_d..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_d_linux ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_cli_linux ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_airgapped_linux ./cmd/airgapped/main.go

.PHONY: mocks
