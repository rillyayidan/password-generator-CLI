# passgen

A customizable password generator CLI written in Go. Uses `crypto/rand` for cryptographically secure randomness.

## Installation

```bash
git clone https://github.com/yourname/passgen
cd passgen
go build -o passgen .
```

Or install directly:

```bash
go install github.com/yourname/passgen@latest
```

## Usage

```
passgen [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-length` | `16` | Password length |
| `-count` | `1` | Number of passwords to generate |
| `-upper` | `true` | Include uppercase letters (A-Z) |
| `-lower` | `true` | Include lowercase letters (a-z) |
| `-numbers` | `true` | Include numbers (0-9) |
| `-symbols` | `true` | Include symbols (!@#...) |
| `-no-ambig` | `false` | Exclude ambiguous characters (0, O, l, 1) |
| `-no-repeats` | `false` | No character appears more than once |
| `-custom` | `""` | Additional characters to include in pool |
| `-separator` | `\n` | Separator between passwords |

### Examples

```bash
# Default: 16-char password with all types
passgen

# 32-char password, generate 5 at once
passgen -length 32 -count 5

# Lowercase + numbers only
passgen -no-symbols -no-upper -length 12

# No ambiguous chars, no repeats
passgen -no-ambig -no-repeats -length 20

# Include custom characters (e.g. currency symbols)
passgen -custom "€£¥" -length 24

# Ten passwords on one line, comma-separated
passgen -length 8 -count 10 -separator ", "

# Pin-style: numbers only, 6 digits
passgen -no-upper -no-lower -no-symbols -length 6
```

## Notes

- Randomness comes from `crypto/rand`, not `math/rand` — safe for real use.
- `-no-repeats` requires the pool to be at least as large as `-length`.
- Strength info (weak/fair/good/strong) is printed to stderr so it doesn't pollute piped output.
- Custom characters are deduplicated before use.
