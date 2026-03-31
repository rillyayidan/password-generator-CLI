package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleFocused   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	styleBlurred   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 3)
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleSubtitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	stylePassword  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("215"))
	styleStrong    = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleGood      = lipgloss.NewStyle().Foreground(lipgloss.Color("148"))
	styleFair      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleWeak      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleOn        = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleOff       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleSelected  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	styleCheckbox  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	charUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charLower   = "abcdefghijklmnopqrstuvwxyz"
	charNumbers = "0123456789"
	charSymbols = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	charAmbig   = "0Ol1"

	minLen = 4
	maxLen = 64
)

// ── Field indices ─────────────────────────────────────────────────────────────

const (
	fieldLength = iota
	fieldUpper
	fieldLower
	fieldNumbers
	fieldSymbols
	fieldNoAmbig
	fieldNoRepeats
	fieldCustom
	fieldGenerate
	fieldCount // total number of fields
)

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	cursor    int
	length    int
	upper     bool
	lower     bool
	numbers   bool
	symbols   bool
	noAmbig   bool
	noRepeats bool
	custom    string
	typingCustom bool

	password string
	strength string
	err      string
}

func initialModel() model {
	return model{
		cursor:  0,
		length:  16,
		upper:   true,
		lower:   true,
		numbers: true,
		symbols: true,
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If typing in custom chars field
		if m.typingCustom {
			switch msg.Type {
			case tea.KeyEnter, tea.KeyEsc:
				m.typingCustom = false
			case tea.KeyBackspace:
				if len(m.custom) > 0 {
					m.custom = m.custom[:len(m.custom)-1]
				}
			default:
				if msg.Type == tea.KeyRunes {
					m.custom += string(msg.Runes)
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j", "tab":
			if m.cursor < fieldCount-1 {
				m.cursor++
			}

		case "shift+tab":
			if m.cursor > 0 {
				m.cursor--
			}

		case "left", "h":
			if m.cursor == fieldLength && m.length > minLen {
				m.length--
				m.password = ""
			}

		case "right", "l":
			if m.cursor == fieldLength && m.length < maxLen {
				m.length++
				m.password = ""
			}

		case " ", "enter":
			switch m.cursor {
			case fieldUpper:
				if !m.upper || m.lower || m.numbers || m.symbols {
					m.upper = !m.upper
					m.password = ""
				}
			case fieldLower:
				if !m.lower || m.upper || m.numbers || m.symbols {
					m.lower = !m.lower
					m.password = ""
				}
			case fieldNumbers:
				if !m.numbers || m.upper || m.lower || m.symbols {
					m.numbers = !m.numbers
					m.password = ""
				}
			case fieldSymbols:
				if !m.symbols || m.upper || m.lower || m.numbers {
					m.symbols = !m.symbols
					m.password = ""
				}
			case fieldNoAmbig:
				m.noAmbig = !m.noAmbig
				m.password = ""
			case fieldNoRepeats:
				m.noRepeats = !m.noRepeats
				m.password = ""
			case fieldCustom:
				m.typingCustom = true
			case fieldGenerate:
				pass, err := generatePassword(m)
				if err != nil {
					m.err = err.Error()
					m.password = ""
				} else {
					m.password = pass
					m.strength = calcStrength(pass)
					m.err = ""
				}
			}
		}
	}
	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("  passgen") + "  " + styleSubtitle.Render("interactive password generator") + "\n\n")

	rows := []string{
		m.rowLength(),
		m.rowToggle(fieldUpper,   m.upper,     "uppercase letters (A–Z)"),
		m.rowToggle(fieldLower,   m.lower,     "lowercase letters (a–z)"),
		m.rowToggle(fieldNumbers, m.numbers,   "numbers (0–9)"),
		m.rowToggle(fieldSymbols, m.symbols,   "symbols (!@#$...)"),
		m.rowToggle(fieldNoAmbig,   m.noAmbig,   "exclude ambiguous chars (0 O l 1)"),
		m.rowToggle(fieldNoRepeats, m.noRepeats, "no repeating characters"),
		m.rowCustom(),
		m.rowGenerate(),
	}

	for _, row := range rows {
		b.WriteString(row + "\n")
	}

	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("  " + styleWeak.Render("✗ "+m.err) + "\n\n")
	} else if m.password != "" {
		b.WriteString("  " + stylePassword.Render(m.password) + "\n")
		b.WriteString("  " + m.strengthBar() + "\n\n")
	}

	b.WriteString("\n" + styleHelp.Render("  ↑/↓ navigate  ·  space/enter select  ·  ←/→ adjust length  ·  q quit"))

	return styleBorder.Render(b.String())
}

func (m model) cursor_str(field int) string {
	if m.cursor == field {
		return styleSelected.Render("▶")
	}
	return "  "
}

func (m model) rowLength() string {
	cur := m.cursor_str(fieldLength)
	label := "length"
	if m.cursor == fieldLength {
		label = styleFocused.Render(label)
	} else {
		label = styleBlurred.Render(label)
	}

	bar := m.lengthBar()
	val := fmt.Sprintf("%d", m.length)
	if m.cursor == fieldLength {
		val = styleFocused.Render(val)
	}
	return fmt.Sprintf("  %s  %-22s  %s  %s", cur, label, bar, val)
}

func (m model) lengthBar() string {
	total := 20
	filled := int(float64(m.length-minLen) / float64(maxLen-minLen) * float64(total))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", total-filled)
	if m.cursor == fieldLength {
		return styleFocused.Render(bar)
	}
	return styleBlurred.Render(bar)
}

func (m model) rowToggle(field int, val bool, label string) string {
	cur := m.cursor_str(field)
	checkbox := "[ ]"
	if val {
		checkbox = styleOn.Render("[✓]")
	} else {
		checkbox = styleOff.Render("[ ]")
	}
	var lbl string
	if m.cursor == field {
		lbl = styleFocused.Render(label)
	} else {
		lbl = styleBlurred.Render(label)
	}
	return fmt.Sprintf("  %s  %s  %s", cur, checkbox, lbl)
}

func (m model) rowCustom() string {
	cur := m.cursor_str(fieldCustom)
	var label string
	if m.cursor == fieldCustom {
		label = styleFocused.Render("custom chars")
	} else {
		label = styleBlurred.Render("custom chars")
	}

	var val string
	if m.typingCustom {
		val = styleFocused.Render("[" + m.custom + "█]")
	} else if m.custom != "" {
		val = styleOn.Render("[" + m.custom + "]")
	} else {
		val = styleOff.Render("[press enter to type]")
	}
	return fmt.Sprintf("  %s  %s  %s", cur, label, val)
}

func (m model) rowGenerate() string {
	cur := m.cursor_str(fieldGenerate)
	var btn string
	if m.cursor == fieldGenerate {
		btn = styleSelected.Render("[ generate password ]")
	} else {
		btn = styleBlurred.Render("[ generate password ]")
	}
	return fmt.Sprintf("\n  %s  %s", cur, btn)
}

func (m model) strengthBar() string {
	score := strengthScore(m.password)
	segs := 4
	labels := []string{"", "weak", "fair", "good", "strong"}
	styles := []*lipgloss.Style{nil, &styleWeak, &styleFair, &styleGood, &styleStrong}

	filled := score
	bar := ""
	for i := 1; i <= segs; i++ {
		if i <= filled {
			bar += styles[score].Render("█")
		} else {
			bar += styleBlurred.Render("░")
		}
	}
	label := ""
	if score > 0 {
		label = "  " + styles[score].Render(labels[score])
	}
	return bar + label
}

// ── Generation logic ──────────────────────────────────────────────────────────

func buildPool(m model) (string, error) {
	var pool strings.Builder
	if m.upper   { pool.WriteString(charUpper) }
	if m.lower   { pool.WriteString(charLower) }
	if m.numbers { pool.WriteString(charNumbers) }
	if m.symbols { pool.WriteString(charSymbols) }
	if m.custom != "" { pool.WriteString(m.custom) }

	result := pool.String()
	if m.noAmbig {
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
		return "", fmt.Errorf("character pool is empty")
	}
	if m.noRepeats && m.length > len([]rune(result)) {
		return "", fmt.Errorf("pool too small (%d chars) for no-repeats with length %d", len([]rune(result)), m.length)
	}
	return result, nil
}

func generatePassword(m model) (string, error) {
	pool, err := buildPool(m)
	if err != nil {
		return "", err
	}
	runes := []rune(pool)
	result := make([]rune, 0, m.length)

	available := make([]rune, len(runes))
	copy(available, runes)

	for i := 0; i < m.length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
		if err != nil {
			return "", err
		}
		idx := n.Int64()
		result = append(result, available[idx])
		if m.noRepeats {
			available = append(available[:idx], available[idx+1:]...)
		}
	}
	return string(result), nil
}

func strengthScore(pass string) int {
	if pass == "" {
		return 0
	}
	score := 0
	if len(pass) >= 8  { score++ }
	if len(pass) >= 16 { score++ }
	if strings.ContainsAny(pass, charUpper) && strings.ContainsAny(pass, charLower) { score++ }
	if strings.ContainsAny(pass, charNumbers) { score++ }
	if strings.ContainsAny(pass, charSymbols) { score++ }

	switch {
	case score <= 1: return 1
	case score <= 2: return 2
	case score <= 3: return 3
	default:         return 4
	}
}

func calcStrength(pass string) string {
	labels := []string{"", "weak", "fair", "good", "strong"}
	return labels[strengthScore(pass)]
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}