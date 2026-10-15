BINARY := nem
PKG := ./cmd/nem
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build rebuild clean test vet fmt

all: build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

rebuild: clean build

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...
