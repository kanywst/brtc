.PHONY: build test format lint vuln clean

BINARY_NAME=brtc
MOD_DIR=github.com/kanywst/brtc

build:
	go build -o $(BINARY_NAME) main.go

test:
	go test -v ./...

format:
	go fmt ./...
	go mod tidy

lint:
	golangci-lint run ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

clean:
	go clean
	rm -f $(BINARY_NAME)
