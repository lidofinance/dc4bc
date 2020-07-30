test:
	@echo "Testing Go packages..."
	@go test ./... -cover

mocks:
	@echo "Regenerate mocks..."
	@go generate ./...

.PHONY: mocks