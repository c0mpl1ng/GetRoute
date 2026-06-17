.PHONY: build build-all test lint clean

MODULE   := GetRoute
BIN_DIR  := bin
CMD_DIR  := ./cmd/getroute
BIN_NAME := GetRoute

build:
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME) $(CMD_DIR)

build-all:
	@mkdir -p $(BIN_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME)-linux-amd64   $(CMD_DIR)
	GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME)-linux-arm64   $(CMD_DIR)
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME)-darwin-amd64  $(CMD_DIR)
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME)-darwin-arm64  $(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Build complete. Binaries in $(BIN_DIR)/"

test:
	go test -v -race -timeout 120s ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)
