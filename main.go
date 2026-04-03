package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	charUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charLower   = "abcdefghijklmnopqrstuvwxyz"
	charNumbers = "0123456789"
	charSymbols = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	charAmbig   = "0Ol1"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type GenerateRequest struct {
	Length    int    `json:"length"`
	Upper     bool   `json:"upper"`
	Lower     bool   `json:"lower"`
	Numbers   bool   `json:"numbers"`
	Symbols   bool   `json:"symbols"`
	NoAmbig   bool   `json:"noAmbig"`
	NoRepeats bool   `json:"noRepeats"`
	Custom    string `json:"custom"`
	Count     int    `json:"count"`
}

type GenerateResponse struct {
	Passwords []string `json:"passwords"`
	Strength  string   `json:"strength"`
	PoolSize  int      `json:"poolSize"`
	Error     string   `json:"error,omitempty"`
}

// ── Core logic ────────────────────────────────────────────────────────────────

func buildPool(req GenerateRequest) (string, error) {
	var pool strings.Builder
	if req.Upper {
		pool.WriteString(charUpper)
	}
	if req.Lower {
		pool.WriteString(charLower)
	}
	if req.Numbers {
		pool.WriteString(charNumbers)
	}
	if req.Symbols {
		pool.WriteString(charSymbols)
	}
	if req.Custom != "" {
		pool.WriteString(req.Custom)
	}

	result := pool.String()

	if req.NoAmbig {
		var filtered strings.Builder
		for _, ch := range result {
			if !strings.ContainsRune(charAmbig, ch) {
				filtered.WriteRune(ch)
			}
		}
		result = filtered.String()
	}

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
	if req.NoRepeats && req.Length > len([]rune(result)) {
		return "", fmt.Errorf("pool too small (%d chars) for no-repeats at length %d", len([]rune(result)), req.Length)
	}
	return result, nil
}

func generateOne(pool string, length int, noRepeats bool) (string, error) {
	runes := []rune(pool)
	available := make([]rune, len(runes))
	copy(available, runes)
	result := make([]rune, 0, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
		if err != nil {
			return "", err
		}
		idx := n.Int64()
		result = append(result, available[idx])
		if noRepeats {
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
		return 1
	case score <= 2:
		return 2
	case score <= 3:
		return 3
	default:
		return 4
	}
}

func strengthLabel(score int) string {
	return []string{"", "weak", "fair", "good", "strong"}[score]
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, GenerateResponse{Error: "method not allowed"})
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, GenerateResponse{Error: "invalid request"})
		return
	}

	if req.Length < 4 {
		req.Length = 4
	}
	if req.Length > 64 {
		req.Length = 64
	}
	if req.Count < 1 {
		req.Count = 1
	}
	if req.Count > 20 {
		req.Count = 20
	}

	if !req.Upper && !req.Lower && !req.Numbers && !req.Symbols && req.Custom == "" {
		req.Lower = true
		req.Numbers = true
	}

	pool, err := buildPool(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, GenerateResponse{Error: err.Error()})
		return
	}

	passwords := make([]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		pass, err := generateOne(pool, req.Length, req.NoRepeats)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, GenerateResponse{Error: err.Error()})
			return
		}
		passwords = append(passwords, pass)
	}

	score := strengthScore(passwords[0])
	writeJSON(w, http.StatusOK, GenerateResponse{
		Passwords: passwords,
		Strength:  strengthLabel(score),
		PoolSize:  len([]rune(pool)),
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/generate", handleGenerate)

	port := "8080"
	fmt.Printf("passgen running at http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

// ── Embedded frontend ─────────────────────────────────────────────────────────

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>passgen</title>
<link href="https://fonts.googleapis.com/css2?family=Share+Tech+Mono&family=DM+Sans:wght@300;400;500&display=swap" rel="stylesheet">
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  :root {
    --bg:       #0d0f12;
    --surface:  #141720;
    --border:   #1e2330;
    --border2:  #2a3045;
    --accent:   #00e5a0;
    --accent2:  #00b37a;
    --text:     #e8eaf0;
    --muted:    #5a6178;
    --muted2:   #8892a8;
    --weak:     #ff4d4d;
    --fair:     #ffaa00;
    --good:     #88cc00;
    --strong:   #00e5a0;
    --mono:     'Share Tech Mono', monospace;
    --sans:     'DM Sans', sans-serif;
  }

  body {
    background: var(--bg);
    color: var(--text);
    font-family: var(--sans);
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem 1rem;
  }

  .container {
    width: 100%;
    max-width: 560px;
  }

  header {
    margin-bottom: 2rem;
  }

  .logo {
    font-family: var(--mono);
    font-size: 1.5rem;
    color: var(--accent);
    letter-spacing: 2px;
  }

  .logo span { color: var(--muted2); font-size: 0.85rem; margin-left: 0.75rem; letter-spacing: 1px; }

  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.75rem;
    margin-bottom: 1rem;
  }

  .card-title {
    font-size: 0.7rem;
    font-weight: 500;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--muted);
    margin-bottom: 1.25rem;
  }

  /* Output */
  .output-wrap {
    position: relative;
    margin-bottom: 1rem;
  }

  .password-display {
    font-family: var(--mono);
    font-size: 1.05rem;
    background: var(--bg);
    border: 1px solid var(--border2);
    border-radius: 8px;
    padding: 1rem 3.5rem 1rem 1rem;
    min-height: 52px;
    word-break: break-all;
    color: var(--accent);
    letter-spacing: 1px;
    line-height: 1.6;
    transition: border-color 0.2s;
  }

  .password-display.placeholder { color: var(--muted); font-size: 0.9rem; letter-spacing: 0; }
  .password-display.flash { border-color: var(--accent); }

  .copy-btn {
    position: absolute;
    top: 50%;
    right: 0.75rem;
    transform: translateY(-50%);
    background: none;
    border: 1px solid var(--border2);
    border-radius: 6px;
    color: var(--muted2);
    font-family: var(--mono);
    font-size: 0.7rem;
    padding: 4px 8px;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
  }
  .copy-btn:hover { border-color: var(--accent); color: var(--accent); }
  .copy-btn.copied { border-color: var(--accent); color: var(--accent); }

  /* Strength bar */
  .strength-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.5rem;
  }

  .strength-segs {
    display: flex;
    gap: 4px;
    flex: 1;
  }

  .seg {
    height: 3px;
    flex: 1;
    border-radius: 2px;
    background: var(--border2);
    transition: background 0.4s;
  }

  .strength-label {
    font-family: var(--mono);
    font-size: 0.7rem;
    letter-spacing: 1px;
    width: 48px;
    text-align: right;
    color: var(--muted);
    transition: color 0.3s;
  }

  .pool-info {
    font-size: 0.72rem;
    color: var(--muted);
    font-family: var(--mono);
  }

  /* Multiple passwords */
  .multi-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 0.75rem;
  }

  .multi-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 0.6rem 0.85rem;
    font-family: var(--mono);
    font-size: 0.85rem;
    color: var(--text);
    word-break: break-all;
    gap: 0.75rem;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: none; } }

  .multi-copy {
    background: none;
    border: none;
    color: var(--muted);
    cursor: pointer;
    font-size: 0.75rem;
    font-family: var(--mono);
    white-space: nowrap;
    padding: 2px 4px;
    flex-shrink: 0;
    transition: color 0.15s;
  }
  .multi-copy:hover { color: var(--accent); }

  /* Controls */
  .field { margin-bottom: 1.25rem; }
  .field:last-child { margin-bottom: 0; }

  .field-label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.8rem;
    color: var(--muted2);
    margin-bottom: 0.5rem;
  }

  .field-label .val {
    font-family: var(--mono);
    color: var(--accent);
    font-size: 0.85rem;
  }

  input[type=range] {
    -webkit-appearance: none;
    width: 100%;
    height: 3px;
    background: var(--border2);
    border-radius: 2px;
    outline: none;
  }
  input[type=range]::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--accent);
    cursor: pointer;
    transition: transform 0.1s;
  }
  input[type=range]::-webkit-slider-thumb:hover { transform: scale(1.2); }

  .toggles {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .toggle {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    cursor: pointer;
    user-select: none;
    font-size: 0.8rem;
    color: var(--muted2);
    transition: all 0.15s;
    background: var(--bg);
  }

  .toggle:hover { border-color: var(--border2); color: var(--text); }

  .toggle.on {
    border-color: var(--accent2);
    color: var(--text);
    background: rgba(0, 229, 160, 0.05);
  }

  .toggle-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
    background: var(--border2);
    flex-shrink: 0;
    transition: background 0.15s;
  }

  .toggle.on .toggle-dot { background: var(--accent); }

  .options-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .custom-input {
    width: 100%;
    background: var(--bg);
    border: 1px solid var(--border2);
    border-radius: 8px;
    padding: 0.6rem 0.85rem;
    color: var(--text);
    font-family: var(--mono);
    font-size: 0.9rem;
    outline: none;
    transition: border-color 0.2s;
  }
  .custom-input:focus { border-color: var(--accent); }
  .custom-input::placeholder { color: var(--muted); }

  .count-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .count-btn {
    width: 28px; height: 28px;
    border-radius: 6px;
    border: 1px solid var(--border2);
    background: none;
    color: var(--muted2);
    font-size: 1rem;
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: all 0.15s;
    flex-shrink: 0;
  }
  .count-btn:hover { border-color: var(--accent); color: var(--accent); }

  .count-val {
    font-family: var(--mono);
    font-size: 0.9rem;
    color: var(--text);
    min-width: 20px;
    text-align: center;
  }

  /* Generate button */
  .gen-btn {
    width: 100%;
    padding: 0.9rem;
    background: var(--accent);
    border: none;
    border-radius: 10px;
    color: #0d0f12;
    font-family: var(--mono);
    font-size: 0.95rem;
    letter-spacing: 2px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s;
    margin-top: 1rem;
  }
  .gen-btn:hover { background: #00ffb3; transform: translateY(-1px); }
  .gen-btn:active { transform: translateY(0); }
  .gen-btn:disabled { background: var(--border2); color: var(--muted); cursor: not-allowed; transform: none; }

  .error-msg {
    font-size: 0.8rem;
    color: var(--weak);
    font-family: var(--mono);
    margin-top: 0.5rem;
    min-height: 1rem;
  }

  /* Divider */
  .divider {
    height: 1px;
    background: var(--border);
    margin: 1.25rem 0;
  }

  footer {
    text-align: center;
    font-size: 0.72rem;
    color: var(--muted);
    font-family: var(--mono);
    margin-top: 1.5rem;
  }
</style>
</head>
<body>
<div class="container">
  <header>
    <div class="logo">passgen<span>// secure password generator</span></div>
  </header>

  <!-- Output card -->
  <div class="card">
    <div class="card-title">output</div>
    <div class="output-wrap">
      <div class="password-display placeholder" id="passDisplay">hit generate to create a password</div>
      <button class="copy-btn" id="copyBtn" onclick="copyMain()">copy</button>
    </div>
    <div class="strength-row">
      <div class="strength-segs">
        <div class="seg" id="seg1"></div>
        <div class="seg" id="seg2"></div>
        <div class="seg" id="seg3"></div>
        <div class="seg" id="seg4"></div>
      </div>
      <div class="strength-label" id="strengthLabel"></div>
    </div>
    <div class="pool-info" id="poolInfo"></div>
    <div class="multi-list" id="multiList"></div>
  </div>

  <!-- Config card -->
  <div class="card">
    <div class="card-title">configure</div>

    <div class="field">
      <div class="field-label">
        <span>length</span>
        <span class="val" id="lenVal">16</span>
      </div>
      <input type="range" id="lengthSlider" min="4" max="64" value="16" step="1"
        oninput="document.getElementById('lenVal').textContent = this.value">
    </div>

    <div class="divider"></div>

    <div class="field">
      <div class="field-label" style="margin-bottom: 0.6rem;">character types</div>
      <div class="toggles">
        <div class="toggle on" id="tUpper" onclick="tog('upper')"><div class="toggle-dot"></div>uppercase A–Z</div>
        <div class="toggle on" id="tLower" onclick="tog('lower')"><div class="toggle-dot"></div>lowercase a–z</div>
        <div class="toggle on" id="tNumbers" onclick="tog('numbers')"><div class="toggle-dot"></div>numbers 0–9</div>
        <div class="toggle on" id="tSymbols" onclick="tog('symbols')"><div class="toggle-dot"></div>symbols !@#...</div>
      </div>
    </div>

    <div class="divider"></div>

    <div class="field">
      <div class="field-label" style="margin-bottom: 0.6rem;">options</div>
      <div class="options-grid">
        <div class="toggle" id="tNoAmbig" onclick="tog('noAmbig')"><div class="toggle-dot"></div>no ambiguous</div>
        <div class="toggle" id="tNoRepeats" onclick="tog('noRepeats')"><div class="toggle-dot"></div>no repeats</div>
      </div>
    </div>

    <div class="divider"></div>

    <div class="field">
      <div class="field-label">custom characters</div>
      <input class="custom-input" id="customChars" type="text" placeholder="e.g.  € £ ¥  or any extra chars">
    </div>

    <div class="divider"></div>

    <div class="field">
      <div class="field-label">
        <span>how many passwords?</span>
      </div>
      <div class="count-row">
        <button class="count-btn" onclick="adjCount(-1)">−</button>
        <span class="count-val" id="countVal">1</span>
        <button class="count-btn" onclick="adjCount(1)">+</button>
      </div>
    </div>
  </div>

  <div class="error-msg" id="errorMsg"></div>
  <button class="gen-btn" id="genBtn" onclick="generate()">GENERATE</button>

  <footer>crypto/rand · go · passgen</footer>
</div>

<script>
  const state = { upper: true, lower: true, numbers: true, symbols: true, noAmbig: false, noRepeats: false };
  let count = 1;

  const charTypes = ['upper','lower','numbers','symbols'];
  const idMap = { upper:'tUpper', lower:'tLower', numbers:'tNumbers', symbols:'tSymbols', noAmbig:'tNoAmbig', noRepeats:'tNoRepeats' };

  function tog(key) {
    if (charTypes.includes(key)) {
      const activeCount = charTypes.filter(k => state[k]).length;
      if (state[key] && activeCount === 1) return;
    }
    state[key] = !state[key];
    document.getElementById(idMap[key]).classList.toggle('on', state[key]);
  }

  function adjCount(d) {
    count = Math.max(1, Math.min(20, count + d));
    document.getElementById('countVal').textContent = count;
  }

  async function generate() {
    const btn = document.getElementById('genBtn');
    btn.disabled = true;
    btn.textContent = 'GENERATING...';
    document.getElementById('errorMsg').textContent = '';

    const body = {
      length:    parseInt(document.getElementById('lengthSlider').value),
      upper:     state.upper,
      lower:     state.lower,
      numbers:   state.numbers,
      symbols:   state.symbols,
      noAmbig:   state.noAmbig,
      noRepeats: state.noRepeats,
      custom:    document.getElementById('customChars').value,
      count:     count,
    };

    try {
      const res = await fetch('/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();

      if (data.error) {
        document.getElementById('errorMsg').textContent = '✗ ' + data.error;
        clearOutput();
      } else {
        renderOutput(data);
      }
    } catch (e) {
      document.getElementById('errorMsg').textContent = '✗ could not reach server';
      clearOutput();
    }

    btn.disabled = false;
    btn.textContent = 'GENERATE';
  }

  function clearOutput() {
    const d = document.getElementById('passDisplay');
    d.textContent = 'hit generate to create a password';
    d.className = 'password-display placeholder';
    document.getElementById('strengthLabel').textContent = '';
    document.getElementById('poolInfo').textContent = '';
    document.getElementById('multiList').innerHTML = '';
    ['seg1','seg2','seg3','seg4'].forEach(id => document.getElementById(id).style.background = '');
  }

  const strengthColors = { weak: '#ff4d4d', fair: '#ffaa00', good: '#88cc00', strong: '#00e5a0' };
  const strengthScores = { weak: 1, fair: 2, good: 3, strong: 4 };

  function renderOutput(data) {
    const pass = data.passwords[0];
    const d = document.getElementById('passDisplay');
    d.textContent = pass;
    d.className = 'password-display flash';
    setTimeout(() => d.classList.remove('flash'), 300);

    const score = strengthScores[data.strength] || 0;
    const color = strengthColors[data.strength] || '#2a3045';
    ['seg1','seg2','seg3','seg4'].forEach((id, i) => {
      document.getElementById(id).style.background = i < score ? color : 'var(--border2)';
    });
    document.getElementById('strengthLabel').textContent = data.strength;
    document.getElementById('strengthLabel').style.color = color;
    document.getElementById('poolInfo').textContent = 'pool: ' + data.poolSize + ' chars';

    const ml = document.getElementById('multiList');
    ml.innerHTML = '';
    if (data.passwords.length > 1) {
      data.passwords.slice(1).forEach((p, i) => {
        const row = document.createElement('div');
        row.className = 'multi-item';
        row.style.animationDelay = (i * 0.04) + 's';
        row.innerHTML = '<span>' + escHtml(p) + '</span><button class="multi-copy" onclick="copyText(this, \'' + escHtml(p) + '\')">copy</button>';
        ml.appendChild(row);
      });
    }
  }

  function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;'); }

  function copyMain() {
    const txt = document.getElementById('passDisplay').textContent;
    if (document.getElementById('passDisplay').classList.contains('placeholder')) return;
    copyText(document.getElementById('copyBtn'), txt);
  }

  function copyText(btn, text) {
    navigator.clipboard.writeText(text).then(() => {
      const orig = btn.textContent;
      btn.textContent = 'copied!';
      btn.classList.add('copied');
      setTimeout(() => { btn.textContent = orig; btn.classList.remove('copied'); }, 1500);
    });
  }

  document.addEventListener('keydown', e => { if (e.key === 'Enter') generate(); });
</script>
</body>
</html>`
