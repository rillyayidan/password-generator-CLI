# passgen

A customizable password generator with a web frontend, written in Go. Uses `crypto/rand` for cryptographically secure randomness. No external dependencies — just the Go standard library.

## Installation

```bash
git clone https://github.com/rillyayidan/password-generator-CLI.git
cd password-generator
go run main.go
```

Then open your browser at **http://localhost:8080**

Or build a binary:

```bash
go build -o passgen .
./passgen        # macOS/Linux
passgen.exe      # Windows
```

## Features

- **Length slider** — 4 to 128 characters
- **Character type toggles** — uppercase, lowercase, numbers, symbols
- **Exclude ambiguous chars** — removes 0, O, l, 1 from the pool
- **No repeating chars** — each character used at most once
- **Custom characters** — add any extra chars to the pool (e.g. €£¥)
- **Batch generate** — up to 20 passwords at once
- **Copy to clipboard** — one click per password
- **Strength indicator** — weak / fair / good / strong
- **Enter key** — triggers generate from anywhere on the page

## How it works

The Go server exposes two endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /` | GET | Serves the web UI |
| `POST /generate` | POST | Generates passwords, returns JSON |

The frontend is embedded directly in `main.go` as a string constant — no separate files, no build step, no npm. One binary runs everything.

## Notes

- Randomness comes from `crypto/rand`, not `math/rand` — safe for real use.
- No repeating chars requires the pool to be at least as large as the chosen length.
- Custom characters are deduplicated before use.
- No external Go dependencies — standard library only.
