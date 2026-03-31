# passgen

A customizable password generator with an interactive TUI, written in Go. Uses `crypto/rand` for cryptographically secure randomness.

## Installation

```bash
git clone https://github.com/yourname/passgen
cd passgen
go mod download github.com/charmbracelet/bubbletea
go mod download github.com/charmbracelet/lipgloss
go mod tidy
go build -o passgen .
```

## Usage

```bash
go run main.go
```

or after building:

```bash
./passgen        # macOS/Linux
passgen.exe      # Windows
```

## Controls

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate between fields |
| `←` / `→` | Adjust password length (4–64) |
| `space` / `enter` | Toggle a checkbox or confirm selection |
| `enter` on custom chars | Start typing custom characters |
| `enter` / `esc` | Stop typing custom characters |
| `enter` on Generate | Generate a new password |
| `q` / `ctrl+c` | Quit |

## Options

| Option | Description |
|--------|-------------|
| Length | Slider from 4 to 64 characters |
| Uppercase (A–Z) | Toggle on/off |
| Lowercase (a–z) | Toggle on/off |
| Numbers (0–9) | Toggle on/off |
| Symbols (!@#...) | Toggle on/off |
| Exclude ambiguous | Removes 0, O, l, 1 from the pool |
| No repeating chars | Each character appears at most once |
| Custom chars | Any extra characters to add to the pool |

At least one of uppercase, lowercase, numbers, or symbols must remain enabled.

## Notes

- Randomness comes from `crypto/rand`, not `math/rand` — safe for real use.
- No repeating chars requires the pool to be at least as large as the chosen length.
- Custom characters are deduplicated before use.
- A color-coded strength bar (weak → fair → good → strong) is shown after each generation.

Example:
│                                                                            │
│     ▶  length    █████████░░░░░░░░░░░  31                                  │
│         [✓]  uppercase letters (A–Z)                                       │
│         [✓]  lowercase letters (a–z)                                       │
│         [✓]  numbers (0–9)                                                 │
│         [✓]  symbols (!@#$...)                                             │
│         [✓]  exclude ambiguous chars (0 O l 1)                             │
│         [ ]  no repeating characters                                       │
│         custom chars  [^*]                                                 │
│                                                                            │
│         [ generate password ]                                              │
│                                                                            │
│     nt.P24bIJ,M6zzu3Gn?BTfgy@B;(Jz&                                        │
│     ████  strong                                                           │
│                                                                            │
│                                                                            │
│     ↑/↓ navigate  ·  space/enter select  ·  ←/→ adjust length  ·  q quit   │
│                                                                            │
