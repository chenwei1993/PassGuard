package main

import "strings"

func csvEscape(v string) string {
	if strings.ContainsAny(v, "\",\n\r") {
		return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
	}
	return v
}

// buildCSV writes the entries as the Electron app exported them.
func buildCSV(entries []Entry) string {
	header := "site,username,password,url,notes,folder"
	var b strings.Builder
	b.WriteString(header)
	for _, e := range entries {
		b.WriteString("\r\n")
		b.WriteString(csvEscape(e.Site))
		b.WriteByte(',')
		b.WriteString(csvEscape(e.Username))
		b.WriteByte(',')
		b.WriteString(csvEscape(e.Password))
		b.WriteByte(',')
		b.WriteString(csvEscape(e.URL))
		b.WriteByte(',')
		b.WriteString(csvEscape(e.Notes))
		b.WriteByte(',')
		b.WriteString(csvEscape(e.Folder))
	}
	return b.String()
}

// parseCSV reads comma-separated values with quoted fields and doubled quotes.
func parseCSV(text string) [][]string {
	text = strings.TrimPrefix(text, "\uFEFF")
	var rows [][]string
	var row []string
	var field strings.Builder
	inQuotes := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inQuotes {
			if c == '"' {
				if i+1 < len(runes) && runes[i+1] == '"' {
					field.WriteRune('"')
					i++
				} else {
					inQuotes = false
				}
			} else {
				field.WriteRune(c)
			}
			continue
		}
		switch c {
		case '"':
			inQuotes = true
		case ',':
			row = append(row, field.String())
			field.Reset()
		case '\n', '\r':
			if c == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
				i++
			}
			row = append(row, field.String())
			field.Reset()
			rows = append(rows, row)
			row = nil
		default:
			field.WriteRune(c)
		}
	}
	if field.Len() > 0 || len(row) > 0 {
		row = append(row, field.String())
		rows = append(rows, row)
	}
	// Drop rows of empty fields.
	out := rows[:0]
	for _, r := range rows {
		empty := true
		for _, f := range r {
			if strings.TrimSpace(f) != "" {
				empty = false
				break
			}
		}
		if !empty {
			out = append(out, r)
		}
	}
	return out
}

var csvAliases = map[string][]string{
	"site":     {"site", "name", "title", "website", "login", "account", "标题", "网站", "名称", "站点"},
	"username": {"username", "user", "login name", "login", "email", "用户", "用户名", "账号", "邮箱"},
	"password": {"password", "pass", "口令", "密码"},
	"url":      {"url", "uri", "website url", "website", "网址", "链接"},
	"notes":    {"notes", "note", "comment", "备注", "说明"},
	"folder":   {"folder", "category", "group", "分组", "文件夹", "分类"},
}

func mapCSVToEntries(rows [][]string) []Entry {
	if len(rows) < 2 {
		return nil
	}
	header := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		header[i] = strings.ToLower(strings.TrimSpace(h))
	}
	index := map[string]int{}
	for key, aliases := range csvAliases {
		index[key] = -1
		for i, h := range header {
			for _, a := range aliases {
				if h == a {
					index[key] = i
				}
			}
		}
	}
	get := func(row []string, key string) string {
		i := index[key]
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	var entries []Entry
	for _, row := range rows[1:] {
		site := get(row, "site")
		if site == "" {
			continue
		}
		folder := get(row, "folder")
		if folder == "" {
			folder = "其他"
		}
		entries = append(entries, Entry{
			Site:     site,
			Username: get(row, "username"),
			Password: get(row, "password"),
			URL:      get(row, "url"),
			Notes:    get(row, "notes"),
			Folder:   folder,
		})
	}
	return entries
}
