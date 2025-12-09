.PHONY: build test clean install fmt lint modern modern-check deps test-coverage

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

modern-check:
	@echo "make: Checking for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

modern:
	@echo "make: Fixing checks for modern Go code..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

deps:
	go mod tidy
	go mod download
