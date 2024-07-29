.PHONY: build fmt clean lint

OS := $(shell go env GOOS)
BUILDCMD=env GOOS=$(OS) GOARCH=arm64 go build -v

build:
	@$(BUILDCMD) -o go-traceroute cmd/go-traceroute/*.go 

fmt:
	@go fmt ./...

clean:
	@go clean ./...
	@rm -rf ./go-traceroute

lint:
	@golangci-lint run ./...