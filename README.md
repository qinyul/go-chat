I’m experimenting with a hybrid gRPC Stream + Websocket setup using Vanguard. By enabling Vanguard’s JSON codec,the request-body encoding and routing between REST and gRPC becomes much more seamless than traditional approaches. It’s been an interesting workflow so far — I’ve got a few more ideas brewing, but I’ll keep them under wraps for now 

# Makefile

This Makefile helps with **protobuf code generation** and running Go programs for this project.

---

## Variables

- `PROTO_DIR` – Directory with `.proto` files (default: `proto/chat/v1`)  
- `OUT_DIR` – Directory for generated Go files (default: `.`)  

---

## Targets

- `all` – Default. Runs `proto` to generate Go code.  
- `proto` – Generates Go code from all `.proto` files. Checks for required plugins.  
- `clean` – Deletes all generated `.pb.go` files.  
- `server` – Runs the Go server (`cmd/server/main.go`).  
- `client` – Runs the Go client (`cmd/client/main.go`).  
- `websocket` – Runs the WebSocket service (`cmd/websocket/main.go`).  

---

## Usage

```bash
# Generate protobuf Go code
make proto

# Run server
make server

# Run client
make client

# Run WebSocket service
make websocket
