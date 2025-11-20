# Directory containing .proto files
PROTO_DIR=proto/chat/v1



# Output directory (usually your Go module root)
OUT_DIR=.

# Paths to protoc plugins (should be in your $PATH)
PROTOC_GEN_GO=$(shell which protoc-gen-go)
PROTOC_GEN_GO_GRPC=$(shell which protoc-gen-go-grpc)

.PHONY: all proto clean

all: proto

# Generate Go code from all proto files
proto:
	@if [ -z "$(PROTOC_GEN_GO)"]; then \
		echo "ERROR: protoc-gen-go not found. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; \
		exit 1; \
	fi
	@if [ -z "$(PROTOC_GEN_GO_GRPC)"]; then \
		echo "ERROR: protoc-gen-go-grpc not found. Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; \
		exit 1; \
	fi
	
	@echo "Generating protobuf files..."
	protoc \
		--go_out=$(OUT_DIR) \
		--go-grpc_out=$(OUT_DIR) \
		--proto_path=$(PROTO_DIR) \
		$(PROTO_DIR)/*.proto

	@echo "Done."

# Remove generated files (optional)
clean:
	find $(OUT_DIR) -name ".pb.go" -type f - delete
	@echo "Cleaned generated files."

server: 
	go run cmd/server/main.go

client: 
	go run cmd/client/main.go

websocket: 
	go run cmd/websocket/main.go