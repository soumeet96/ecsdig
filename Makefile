BINARY=ecsdig
VERSION=$(shell cat pkg/version/version.go | grep Version | cut -d'"' -f2)

.PHONY: build test vet lint clean check

build:
	go build -ldflags="-s -w" -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: vet test

clean:
	rm -f $(BINARY)

install:
	go install .
