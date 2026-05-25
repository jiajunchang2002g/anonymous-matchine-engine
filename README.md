# Matching Engine Skeleton

A minimal Go project with:

- a TCP server
- a CLI client
- a basic engine package
- an optional stress test client

This is a small starting point for a trading system or matching engine prototype.

---

## Project layout

```text
matching-engine/
├── go.mod
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── client/
│   │   └── main.go
│   └── stress/
│       └── main.go
└── internal/
    └── engine/
        └── engine.go
