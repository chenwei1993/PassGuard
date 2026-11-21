package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed assets/locales/zh-CN.json assets/locales/en.json
var localeFS embed.FS

// localeMessages holds the app's own messages, keyed by flat keys such as
// "vault.title". They are the catalogs the Electron app used, so the words
// stay the same.
var localeMessages = map[string]map[string]string{}

func init() {
	for _, tag := range []string{"zh-CN", "en"} {
		raw, err := localeFS.ReadFile("assets/locales/" + tag + ".json")
		if err != nil {
			continue
		}
		m := map[string]string{}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		localeMessages[tag] = m
	}
}

// localeFile maps a BCP 47 tag to the catalog that holds it.
func localeFile(tag string) string {
	if strings.HasPrefix(strings.ToLower(tag), "zh") {
		return "zh-CN"
	}
	return "en"
}

// tr says key in locale, with the Chinese catalog as the last fallback.
func tr(locale, key string) string {
	if m := localeMessages[localeFile(locale)]; m != nil {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if m := localeMessages["zh-CN"]; m != nil {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return key
}

// trf says key and puts args in its {placeholders}.
func trf(locale, key string, args map[string]any) string {
	s := tr(locale, key)
	for k, v := range args {
		s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprint(v))
	}
	return s
}

func (a *App) t(key string) string { return tr(a.locale, key) }

func (a *App) tf(key string, args map[string]any) string {
	return trf(a.locale, key, args)
}
