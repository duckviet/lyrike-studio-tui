package tui

import (
	"sort"
	"strings"
)

// fuzzyRank filters items to those whose hay(item) matches q, ordered by score
// (best first, stable). An empty q keeps every item in its original order.
func fuzzyRank[T any](items []T, q string, hay func(T) string) []T {
	type scored struct {
		item  T
		score int
	}
	var ms []scored
	for _, it := range items {
		if s, ok := fuzzyScore(q, hay(it)); ok {
			ms = append(ms, scored{it, s})
		}
	}
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].score > ms[j].score })
	out := make([]T, len(ms))
	for i, m := range ms {
		out[i] = m.item
	}
	return out
}

// fuzzyScore reports whether pattern matches text as a subsequence, with a
// score that rewards consecutive runs and word-boundary hits so the most
// relevant matches sort first. An empty pattern matches everything.
func fuzzyScore(pattern, text string) (int, bool) {
	if pattern == "" {
		return 0, true
	}
	p := strings.ToLower(pattern)
	t := strings.ToLower(text)

	score := 0
	ti := 0
	last := -2
	for pi := 0; pi < len(p); pi++ {
		c := p[pi]
		found := false
		for ; ti < len(t); ti++ {
			if t[ti] != c {
				continue
			}
			score++
			if ti == last+1 {
				score += 6 // consecutive
			}
			if ti == 0 || isBoundary(t[ti-1]) {
				score += 10 // word boundary
			}
			last = ti
			ti++
			found = true
			break
		}
		if !found {
			return 0, false
		}
	}
	// Prefer shorter, tighter matches.
	score -= len(t) / 12
	return score, true
}

func isBoundary(b byte) bool {
	switch b {
	case ' ', '-', '_', '.', '/', ':':
		return true
	}
	return false
}
