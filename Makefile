.PHONY: build test clean install fmt lint fix fix-check deps test-coverage

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

fix-check:
	@echo "make: Checking go fix..."
	@go fix -diff ./...

fix:
	@echo "make: Applying go fix..."
	@go fix ./...

deps:
	go mod tidy
	go mod download
