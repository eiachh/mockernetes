# Mockernetes

A deliberately minimal, fake Kubernetes API server.

## Scope

- Speaks real TLS
- Responds with correctly shaped Kubernetes API objects
- Allows kubectl to talk to it
- Understand kubectl communication, discovery, resource creation/listing, TLS auth

## Limitations

- Not a full Kubernetes implementation
- Small, readable, extensible

## How to run

Build the server: `go build ./cmd/apiserver`

Run the server: `./apiserver`

The server listens on :8080 with basic HTTP routes for health, ready, and namespace operations.

Generate certs and kubeconfig: `./generate-certs.sh` (edit kubeconfig port if needed)

Use with: `kubectl --kubeconfig=./kubeconfig get ns` (after server runs on 8443 with TLS)