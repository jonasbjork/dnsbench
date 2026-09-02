BINARY_NAME := dnsbench
OUTPUT_DIR  := output
SRC_DIR     := ./src

.PHONY: all build test clean

all: test build

test:
	go test ./test/... ./internal/...

build:
	mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(SRC_DIR)

clean:
	rm -rf $(OUTPUT_DIR)
