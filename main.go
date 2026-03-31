package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
)

const (
	charUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charLower   = "abcdefghijklmnopqrstuvwxyz"
	charNumbers = "0123456789"
	charSymbols = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	charAmbig   = "0Ol1"
)

type Config struct {
	length     int
	count      int
	upper      bool
	lower      bool
	numbers    bool
	symbols    bool
	noAmbig    bool
	noRepeats  bool
	custom     string
	separator  string
}

func buildPool(cfg Config) (string, error) {
	var pool strings.Builder

	if cfg.upper {
		pool.WriteString(charUpper)
	}
	if cfg.lower {
		pool.WriteString(charLower)
	}
	if cfg.numbers {
		pool.WriteString(charNumbers)
	}
	if cfg.symbols {
		pool.WriteString(charSymbols)
	}
	if cfg.custom != "" {
		pool.WriteString(cfg.custom)
	}

	result := pool.String()

	if cfg.noAmbig {
		var filtered strings.Builder
		for _, ch := range result {
			if !strings.ContainsRune(charAmbig, ch) {
				filtered.WriteRune(ch)
			}
		}
		result = filtered.String()
	}

	// Deduplicate
	seen := make(map[rune]bool)
	var deduped strings.Builder
	for _, ch := range result {
		if !seen[ch] {
			seen[ch] = true
			deduped.WriteRune(ch)
		}
	}
	result = deduped.String()

	if result == "" {
		return "", fmt.Errorf("character pool is empty — enable at least one character type")
	}

	if cfg.noRepeats && cfg.length > len([]rune(result)) {
		return "", fmt.Errorf(
			"cannot generate a %d-char password with no repeats: pool only has %d unique chars",
			cfg.length, len([]rune(result)),
		)
	}

	return result, nil
}

func generate(cfg Config, pool string) (string, error) {
	runes := []rune(pool)
	result := make([]rune, 0, cfg.length)

	available := make([]rune, len(runes))
	copy(available, runes)

	for i := 0; i < cfg.length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
		if err != nil {
			return "", fmt.Errorf("random generation failed: %w", err)
		}
		idx := n.Int64()
		result = append(result, available[idx])

		if cfg.noRepeats {
			available = append(available[:idx], available[idx+1:]...)
		}
	}

	return string(result), nil
}

func strengthLabel(pass string, pool string) string {
	score := 0
	if len(pass) >= 8 {
		score++
	}
	if len(pass) >= 16 {
		score++
	}
	if strings.ContainsAny(pass, charUpper) && strings.ContainsAny(pass, charLower) {
		score++
	}
	if strings.ContainsAny(pass, charNumbers) {
		score++
	}
	if strings.ContainsAny(pass, charSymbols) {
		score++
	}

	switch {
	case score <= 1:
		return "weak"
	case score <= 2:
		return "fair"
	case score <= 3:
		return "good"
	default:
		return "strong"
	}
}

func printUsageExtended() {
	fmt.Println(`passgen — a customizable password generator

USAGE:
  passgen [flags]

FLAGS:`)
	flag.PrintDefaults()
	fmt.Println(`
EXAMPLES:
  passgen
      Generate one 16-char password with all character types

  passgen -length 32 -count 5
      Generate five 32-char passwords

  passgen -no-symbols -no-upper -length 12
      Lowercase + numbers only, 12 chars

  passgen -no-ambig -no-repeats -length 20
      No ambiguous chars (0Ol1), no repeated characters

  passgen -custom "€£¥" -length 24
      Include custom characters in the pool

  passgen -length 8 -count 10 -separator ", "
      Ten short passwords on one line, comma-separated`)
}

func main() {
	cfg := Config{}

	flag.IntVar(&cfg.length, "length", 16, "password length")
	flag.IntVar(&cfg.count, "count", 1, "number of passwords to generate")
	flag.BoolVar(&cfg.upper, "upper", true, "include uppercase letters (A-Z)")
	flag.BoolVar(&cfg.lower, "lower", true, "include lowercase letters (a-z)")
	flag.BoolVar(&cfg.numbers, "numbers", true, "include numbers (0-9)")
	flag.BoolVar(&cfg.symbols, "symbols", true, "include symbols (!@#...)")
	flag.BoolVar(&cfg.noAmbig, "no-ambig", false, "exclude ambiguous characters (0, O, l, 1)")
	flag.BoolVar(&cfg.noRepeats, "no-repeats", false, "no character appears more than once")
	flag.StringVar(&cfg.custom, "custom", "", "additional characters to include in the pool")
	flag.StringVar(&cfg.separator, "separator", "\n", "separator between passwords (default: newline)")

	flag.Usage = printUsageExtended
	flag.Parse()

	if cfg.length < 1 {
		fmt.Fprintln(os.Stderr, "error: length must be at least 1")
		os.Exit(1)
	}
	if cfg.count < 1 {
		fmt.Fprintln(os.Stderr, "error: count must be at least 1")
		os.Exit(1)
	}
	if !cfg.upper && !cfg.lower && !cfg.numbers && !cfg.symbols && cfg.custom == "" {
		fmt.Fprintln(os.Stderr, "error: at least one character type must be enabled")
		os.Exit(1)
	}

	pool, err := buildPool(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	passwords := make([]string, 0, cfg.count)
	for i := 0; i < cfg.count; i++ {
		pass, err := generate(cfg, pool)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		passwords = append(passwords, pass)
	}

	if cfg.count == 1 {
		pass := passwords[0]
		fmt.Println(pass)
		fmt.Fprintf(os.Stderr, "strength: %s | length: %d | pool: %d chars\n",
			strengthLabel(pass, pool), len([]rune(pass)), len([]rune(pool)))
	} else {
		fmt.Print(strings.Join(passwords, cfg.separator))
		fmt.Println()
	}
}
