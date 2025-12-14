# Remote Shell RPC

A distributed Remote Shell System that enables multiple clients to execute shell commands on a remote server using gRPC (Google Remote Procedure Call) framework in Go

At a high level:
- **Server** exposes a gRPC service `ShellService` over TCP
- **Client** connects to the server, creates a session, and provides an **interactive CLI** `remote>` to run commands
- Command results can be returned either as a full response or streamed in real time

## 2. Requirements

### Runtime requirements (Server host)
- OS with a POSIX shell available at **`/bin/bash`**
- Permission to spawn processes (server runs commands under the server process user)
- Network: open a TCP port (default **50051**) for incoming gRPC connections


### Build / run-from-source requirements
- Install **Go**
- **Protocol Buffers compiler (`protoc`)** is installed
- Go protobuf plugins (installable via Makefile):
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

## Usage

### Install / Setup 
```bash
# 1) Clone
git clone <REPO_URL>
cd Remote-Shell-RPC-main

# 2) Download module and library dependencies
sudo apt install -y protobuf-compiler
go mod download

# 3) Install protobuf Go plugins
make install-proto-tools

# 4) Make sure protoc exists
protoc --version

# 5) Generate the gRPC/protobuf Go stubs (required before any build/run)
make proto
```

### Using Makefile (build + run)

Important: these targets create binaries in `./bin/`

Run server + client with default settings

```bash
# Build server & client
make build

# Run server (localhost:50051 by default)
make run-server

# Run client (connects to localhost:50051 by default)
make run-client
```

Run using the provided YAML configs

```bash
make run-server-config
make run-client-config
```

Cleaning artifacts

```bash
make clean
# removes ./bin and proto/*.pb.go
```

## If binaries already exist

If `bin/server` and `bin/client` are already built (e.g., from `make build`)

Start the server

```bash
./bin/server -host 0.0.0.0 -port 50051 -log-level info
# or:
./bin/server -config configs/server.yaml
```

Start the client

```bash
./bin/client -host <SERVER_IP> -port 50051 -client-id <custom_id> -log-level warn
# or:
./bin/client -config configs/client.yaml
```

Once connected, we can run commands at the prompt

```bash
remote> pwd
remote> ls -la
remote> cd /tmp
remote> whoami
```
## Features

- **Multi-client Support**: Handle multiple concurrent client connections

- **Real-time Streaming**: Stream command output in real-time

- **Session Management**: Each client gets an isolated session with its own working directory

- **Interactive Shell**: User-friendly command-line interface

- **Configurable**: YAML-based configuration for both server and client
