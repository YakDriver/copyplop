.PHONY: build test clean install fmt lint fix fix-check prereq-go-fix deps test-coverage

build:
	go build -o bin/copyplop .

test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

install:
	go install .

fmt:
	go fmt ./...

lint:
	golangci-lint run

# The analyzer-based go fix and its -diff flag require Go 1.26+, which is newer
# than the go directive in go.mod.
GO_FIX_MIN_VERSION := 1.26

prereq-go-fix:
	@v=$$(go env GOVERSION | sed 's/^go//'); \
	if [ "$$(printf '%s\n' "$(GO_FIX_MIN_VERSION)" "$$v" | sort -V | head -n1)" != "$(GO_FIX_MIN_VERSION)" ]; then \
		echo "make: go fix targets require Go $(GO_FIX_MIN_VERSION) or later (found $$v)" >&2; \
		exit 1; \
	fi

fix-check: prereq-go-fix
	@echo "make: Checking go fix..."
	@go fix -diff ./...

fix: prereq-go-fix
	@echo "make: Applying go fix..."
	@go fix ./...

deps:
	go mod tidy
	go mod download
