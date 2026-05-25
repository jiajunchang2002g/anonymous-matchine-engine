# Matching Engine in Go

A minimal Go project with:

- a TCP server
- a CLI client
- a basic engine package
- an optional stress test client

This is a small starting point for a trading system or matching engine prototype.

## Stress Test Results

<img width="265" height="269" alt="image" src="https://github.com/user-attachments/assets/dab362f1-e902-411d-9cb9-01d8f07ae303" />

**Machine** : AMD Ryzen 5 5600 6-Core, Windows 11

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
