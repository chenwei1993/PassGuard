package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

const (
	charsUpper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsLower  = "abcdefghijklmnopqrstuvwxyz"
	charsNumber = "0123456789"
	charsSymbol = "!@#$%^&*()_+-=[]{};:,.<>?~"
)

var similarRunes = "0O1lI"
var ambiguousRunes = "\"'`\\|"

// secureIndex returns a uniformly random index in [0, n).
func secureIndex(n int) int {
	if n <= 1 {
		return 0
	}
	var buf [4]byte
	limit := uint64(math.MaxUint32+1) - uint64(math.MaxUint32+1)%uint64(n)
	for {
		_, _ = rand.Read(buf[:])
		v := uint64(binary.LittleEndian.Uint32(buf[:]))
		if v < limit {
			return int(v % uint64(n))
		}
	}
}

func pickRune(charset []rune) rune {
	return charset[secureIndex(len(charset))]
}

func buildCharset(upper, lower, numbers, symbols, excludeSimilar, excludeAmbiguous bool) []rune {
	var b strings.Builder
	if upper {
		b.WriteString(charsUpper)
	}
	if lower {
		b.WriteString(charsLower)
	}
	if numbers {
		b.WriteString(charsNumber)
	}
	if symbols {
		b.WriteString(charsSymbol)
	}
	s := b.String()
	if excludeSimilar {
		s = strings.Map(func(r rune) rune {
			if strings.ContainsRune(similarRunes, r) {
				return -1
			}
			return r
		}, s)
	}
	if excludeAmbiguous {
		s = strings.Map(func(r rune) rune {
			if strings.ContainsRune(ambiguousRunes, r) {
				return -1
			}
			return r
		}, s)
	}
	return []rune(s)
}

func generateRandom(length int, upper, lower, numbers, symbols, excludeSimilar, excludeAmbiguous, excludeDuplicate bool) string {
	charset := buildCharset(upper, lower, numbers, symbols, excludeSimilar, excludeAmbiguous)
	if len(charset) == 0 || length <= 0 {
		return ""
	}
	var out []rune
	var prev rune
	for i := 0; i < length; i++ {
		c := pickRune(charset)
		if excludeDuplicate {
			for tries := 0; c == prev && tries < 20; tries++ {
				c = pickRune(charset)
			}
		}
		out = append(out, c)
		prev = c
	}
	return string(out)
}

var wordlist = []string{
	"red", "blue", "green", "sun", "moon", "star", "fire", "lake", "wind", "rain",
	"ocean", "river", "cloud", "snow", "tree", "leaf", "stone", "mountain", "forest", "desert",
	"tiger", "eagle", "dragon", "fox", "wolf", "bear", "falcon", "otter", "koala", "raven",
	"silver", "golden", "amber", "crimson", "violet", "azure", "jade", "pearl", "cobalt", "emerald",
	"orbit", "comet", "pixel", "quantum", "vertex", "nebula", "photon", "cipher", "rune", "vault",
}

func generateMemorable(length int, upper, numbers bool) string {
	wordCount := max(3, int(math.Round(float64(length)/6)))
	words := make([]string, 0, wordCount)
	for i := 0; i < wordCount; i++ {
		words = append(words, wordlist[secureIndex(len(wordlist))])
	}
	seps := []string{"-", "_", ".", "!"}
	var b strings.Builder
	b.WriteString(words[0])
	for i := 1; i < len(words); i++ {
		b.WriteString(seps[secureIndex(len(seps))])
		b.WriteString(words[i])
	}
	if numbers {
		b.WriteString(fmt.Sprint(secureIndex(10)))
	}
	out := b.String()
	if upper && len(out) > 0 {
		pos := secureIndex(len(out))
		out = out[:pos] + strings.ToUpper(out[pos:pos+1]) + out[pos+1:]
	}
	if len(out) > length {
		out = out[:length]
	}
	return out
}

func generatePin(length int) string {
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteByte(byte('0' + secureIndex(10)))
	}
	return b.String()
}

func entropyBits(password string) float64 {
	pool := 0
	if strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		pool += 26
	}
	if strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		pool += 26
	}
	if strings.ContainsAny(password, "0123456789") {
		pool += 10
	}
	for _, r := range password {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			pool += 33
			break
		}
	}
	if pool == 0 || password == "" {
		return 0
	}
	return float64(len([]rune(password))) * math.Log2(float64(pool))
}

// Strength is what the views show of a password.
type Strength struct {
	Score   int
	Entropy float64
	Bars    int
	Level   string // breached | weak | medium | strong | very-strong
}

func strengthFor(password string, breached bool) Strength {
	if breached {
		return Strength{Score: 20, Bars: 2, Level: "breached"}
	}
	e := entropyBits(password)
	score := int(math.Round((e - 20) * 1.25))
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	bars := 1
	switch {
	case score >= 80:
		bars = 5
	case score >= 60:
		bars = 4
	case score >= 40:
		bars = 3
	case score >= 20:
		bars = 2
	}
	level := "weak"
	switch {
	case score >= 80:
		level = "very-strong"
	case score >= 60:
		level = "strong"
	case score >= 40:
		level = "medium"
	}
	return Strength{Score: score, Entropy: e, Bars: bars, Level: level}
}

func (a *App) strengthLabel(s Strength) string {
	switch s.Level {
	case "breached":
		return a.t("issue.breached")
	case "very-strong":
		return a.t("strength.veryStrong")
	case "strong":
		return a.t("strength.strong")
	case "medium":
		return a.t("strength.medium")
	default:
		return a.t("strength.weak")
	}
}

func (a *App) crackTime(password string) string {
	e := entropyBits(password)
	if password == "" {
		return "—"
	}
	seconds := math.Pow(2, e) / 2 / 1e12
	if seconds < 1 {
		return a.t("gen.crack.instant")
	}
	type unit struct {
		mult float64
		key  string
	}
	units := []unit{
		{60, "gen.crack.seconds"}, {60, "gen.crack.minutes"}, {24, "gen.crack.hours"},
		{365, "gen.crack.days"}, {1e3, "gen.crack.years"}, {1e3, "gen.crack.millennia"}, {1e3, "gen.crack.millionYears"},
	}
	v := seconds
	key := "gen.crack.seconds"
	for _, u := range units {
		if v < u.mult {
			key = u.key
			break
		}
		v /= u.mult
		key = u.key
	}
	return a.tf(key, map[string]any{"v": groupedNumber(int64(math.Round(v)))})
}

// groupedNumber writes an integer with commas, as toLocaleString did.
func groupedNumber(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}
