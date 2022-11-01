.ONESHELL:

LDFLAGS := -ldflags="-X 'github.com/neutron-org/neutron/pkg/wc_roration.Profile=test'" 

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
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_d_darwin ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_cli_darwin ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	@echo "WARNING: CGO_ENABLED=0 is requered to set for airgapped"
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_airgapped_darwin ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_prysm_compatibility_checker_darwin ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_dkg_reinitializer_darwin ./cmd/dkg_reinitializer/

build-linux:
	@echo "Building dc4bc_d..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_d_linux ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_cli_linux ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	@echo "WARNING: CGO_ENABLED=0 is requered to set for airgapped"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_airgapped_linux ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_prysm_compatibility_checker_linux ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dc4bc_dkg_reinitializer_linux ./cmd/dkg_reinitializer/

build:
	@echo "Building dc4bc_d..."
	go build $(LDFLAGS) -o dc4bc_d ./cmd/dc4bc_d/
	@echo "Building dc4bc_cli..."
	go build $(LDFLAGS) -o dc4bc_cli ./cmd/dc4bc_cli/
	@echo "Building dc4bc_airgapped..."
	@echo "WARNING: CGO_ENABLED=0 is requered to set for airgapped"
	go build $(LDFLAGS) -o dc4bc_airgapped ./cmd/airgapped/
	@echo "Building dc4bc_prysm_compatibility_checker..."
	go build $(LDFLAGS) -o dc4bc_prysm_compatibility_checker ./cmd/prysm_compatibility_checker/
	@echo "Building dkg_reinitializer..."
	go build $(LDFLAGS) -o dc4bc_dkg_reinitializer ./cmd/dkg_reinitializer/


%-prod: LDFLAGS := -ldflags="-X 'github.com/neutron-org/neutron/pkg/wc_roration.Profile=production'" 
build-prod: build

build-linux-prod: build-linux

build-darwin-prod: build-darwin

.PHONY: mocks
