.PHONY: build test fmt vet lint clean generate

# Build the ixian binary
build:
	go build -o ixian ./cmd/ixian/

# Run all tests (parallel with race detector)
test:
	go test ./cmd/... ./internal/... -race -parallel 4 -v

# Run tests quickly without race detector
test-fast:
	go test ./cmd/... ./internal/... -parallel 4 -v

# Check formatting
fmt:
	gofmt -l $$(find . -name '*.go' -not -path './vendor/*' -not -path './output/*')

# Fix formatting
fmt-fix:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './output/*')

# Run go vet
vet:
	go vet ./cmd/... ./internal/...

# Run all checks (fmt + vet + test)
check: fmt vet test

# Generate code from the petstore example spec
generate: build
	./ixian --spec testdata/petstore.yaml --output output/

# Clean build artifacts and generated output
clean:
	rm -f ixian
	rm -rf output/
