package main

import "time"

// Issue is one problem the security check found in an entry.
type Issue struct {
	ID    string
	Type  string // breached | weak | duplicate | expired
	Entry *Entry
	Risk  string // high | medium | low
}

const dayMs = int64(24 * 60 * 60 * 1000)

// computeIssues derives the problems from the entries, in the same order of
// priority the Electron app used: breached, weak, duplicate, expired.
func computeIssues(entries []Entry) []Issue {
	byPassword := map[string][]int{}
	for i := range entries {
		byPassword[entries[i].Password] = append(byPassword[entries[i].Password], i)
	}
	now := time.Now().UnixMilli()
	list := make([]Issue, 0, len(entries))
	for i := range entries {
		e := &entries[i]
		s := strengthFor(e.Password, false)
		duplicated := len(byPassword[e.Password]) >= 2
		expired := false
		if e.UpdatedAt != "" {
			if t, err := time.Parse("2006-01-02", e.UpdatedAt); err == nil {
				expired = now-t.UnixMilli() > 365*dayMs
			}
		}
		switch {
		case e.Breached:
			list = append(list, Issue{ID: e.ID, Type: "breached", Entry: e, Risk: "high"})
		case s.Score < 40:
			risk := "medium"
			if s.Score < 20 {
				risk = "high"
			}
			list = append(list, Issue{ID: e.ID, Type: "weak", Entry: e, Risk: risk})
		case duplicated:
			list = append(list, Issue{ID: e.ID, Type: "duplicate", Entry: e, Risk: "medium"})
		case expired:
			list = append(list, Issue{ID: e.ID, Type: "expired", Entry: e, Risk: "low"})
		}
	}
	return list
}

func issueStats(issues []Issue) map[string]int {
	stats := map[string]int{"weak": 0, "duplicate": 0, "breached": 0, "expired": 0}
	for _, i := range issues {
		stats[i.Type]++
	}
	return stats
}

// securityScore is the 0–100 score the security page shows.
func securityScore(issues []Issue) int {
	if len(issues) == 0 {
		return 100
	}
	s := issueStats(issues)
	score := 100 - s["weak"]*4 - s["duplicate"]*3 - s["breached"]*12 - s["expired"]*2
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}
