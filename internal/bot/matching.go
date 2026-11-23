package bot

// sharedInterests returns the interests present in both lists, preserving the
// order of the first list and removing duplicates.
func sharedInterests(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, v := range b {
		set[v] = true
	}
	seen := make(map[string]bool, len(a))
	var out []string
	for _, v := range a {
		if set[v] && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// compatible reports whether two profiles may be paired given each side's
// preferred gender (empty string means "anyone").
func compatible(meGender, myWant, candGender, candWant string) bool {
	if myWant != "" && candGender != myWant {
		return false
	}
	if candWant != "" && meGender != candWant {
		return false
	}
	return true
}

// candidate is the minimal view of a waiting user used by the matcher.
type candidate struct {
	ID        int64
	Gender    string
	Interests []string
	Want      string
}

// pickBest chooses the candidate with the most shared interests. Ties are
// broken by the lowest user id so matching is deterministic.
func pickBest(meInterests []string, candidates []candidate) (candidate, bool) {
	if len(candidates) == 0 {
		return candidate{}, false
	}
	best := candidates[0]
	bestScore := len(sharedInterests(meInterests, best.Interests))
	for _, c := range candidates[1:] {
		score := len(sharedInterests(meInterests, c.Interests))
		if score > bestScore || (score == bestScore && c.ID < best.ID) {
			best, bestScore = c, score
		}
	}
	return best, true
}
