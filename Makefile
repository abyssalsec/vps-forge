VERSION := $(shell cat VERSION)
BINARY := bin/vps-forge
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test race fmt fmt-check vet check clean

build:
	mkdir -p bin
	go build \
		-trimpath \
		-ldflags "$(LDFLAGS)" \
		-o $(BINARY) \
		./cmd/vps-forge

test:
	go test ./...

race:
	go test -race ./...

fmt:
	gofmt -w cmd internal

fmt-check:
	@files="$$(gofmt -l cmd internal)"; \
	if [ -n "$$files" ]; then \
		echo "ERROR: gofmt required:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet:
	go vet ./...

check: fmt-check vet test build

clean:
	rm -rf bin dist
