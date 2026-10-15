BINARY := nem
PKG := ./cmd/nem
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build rebuild clean install test vet fmt

all: build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

rebuild: clean build

# Install beside the nem already on PATH, so an update replaces it in place;
# with none there, into $(PREFIX)/bin. Set BINDIR to choose another directory.
PREFIX ?= $(HOME)/.local
CURRENT_NEM := $(shell command -v $(BINARY) 2>/dev/null)
BINDIR ?= $(if $(CURRENT_NEM),$(dir $(CURRENT_NEM)),$(PREFIX)/bin)

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	go fmt ./...
install: build
	mkdir -p $(DESTDIR)$(BINDIR)
	cp $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

