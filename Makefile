# Makefile for common dev tasks; run `make <target>` from the repo root.
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build test race cover fmt vet run tidy generate

build: generate
	@mkdir -p bin
	go build ./...
	go build -o bin/memories ./cmd/memories

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -cover ./...

fmt:
	gofmt -w $(GO_FILES)

vet:
	go vet ./...

run: build
	./bin/memories

tidy:
	go mod tidy

generate:
	go tool templ generate
