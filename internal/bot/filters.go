package bot

import (
	"fmt"
	"strconv"
	"strings"

	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

// filtersAllow applies the searcher's age, country and region preferences.
func (b *Bot) filtersAllow(me store.User, cand store.User) bool {
	if me.WantAgeMin > 0 && cand.Age > 0 && cand.Age < me.WantAgeMin {
		return false
	}
	if me.WantAgeMax > 0 && cand.Age > 0 && cand.Age > me.WantAgeMax {
		return false
	}
	if me.WantCountry != "" && !strings.EqualFold(strings.TrimSpace(cand.Country), me.WantCountry) {
		return false
	}
	switch me.WantRegion {
	case "india":
		if !isIndiaCountry(cand.Country) {
			return false
		}
	case "foreign":
		if isIndiaCountry(cand.Country) {
			return false
		}
	}
	return true
}

// isIndiaCountry reports whether a free-form country string means India.
func isIndiaCountry(c string) bool {
	c = strings.TrimSpace(c)
	return strings.EqualFold(c, "india") || strings.EqualFold(c, "in")
}

// cmdFind sets age and country preferences for matching.
func (b *Bot) cmdFind(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	args = strings.TrimSpace(args)
	if args == "" {
		b.states.setStep(uid, "find_age")
		b.sendText(uid, FindAgePrompt, nil)
		return
	}
	fields := strings.Fields(args)
	minAge, maxAge := 0, 0
	if len(fields) > 0 {
		lo, hi, ok := parseAgeRange(fields[0])
		if !ok {
			b.sendText(uid, FindAgeInvalid, nil)
			return
		}
		minAge, maxAge = lo, hi
	}
	country := ""
	if len(fields) > 1 {
		country = strings.Join(fields[1:], " ")
	}
	b.st.SetFilters(uid, minAge, maxAge, country)
	b.states.clear(uid)
	b.sendText(uid, fmt.Sprintf(FindSaved, b.describeFilters(minAge, maxAge, country)), nil)
}

func parseAgeRange(s string) (int, int, bool) {
	if !strings.Contains(s, "-") {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, false
		}
		return n, n, true
	}
	parts := strings.SplitN(s, "-", 2)
	lo, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	hi, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

func (b *Bot) findAge(m *tgbotapi.Message, st *state) {
	n, err := strconv.Atoi(strings.TrimSpace(m.Text))
	if err != nil || n < b.cfg.MinAge || n > b.cfg.MaxAge {
		b.sendText(m.From.ID, fmt.Sprintf(FindAgeInvalidRange, b.cfg.MinAge, b.cfg.MaxAge), nil)
		return
	}
	st.AgeMin = n
	b.states.setStep(m.From.ID, "find_max")
	b.sendText(m.From.ID, FindMaxPrompt, nil)
}

func (b *Bot) findMax(m *tgbotapi.Message, st *state) {
	n, err := strconv.Atoi(strings.TrimSpace(m.Text))
	if err != nil || n < st.AgeMin || n > b.cfg.MaxAge {
		b.sendText(m.From.ID, fmt.Sprintf(FindAgeInvalidRange, st.AgeMin, b.cfg.MaxAge), nil)
		return
	}
	st.AgeMax = n
	b.states.setStep(m.From.ID, "find_country")
	b.sendText(m.From.ID, FindCountryPrompt, nil)
}

func (b *Bot) findCountry(m *tgbotapi.Message, st *state) {
	country := strings.TrimSpace(m.Text)
	if country == "-" || strings.EqualFold(country, "any") {
		country = ""
	}
	b.st.SetFilters(m.From.ID, st.AgeMin, st.AgeMax, country)
	b.states.clear(m.From.ID)
	b.sendText(m.From.ID, fmt.Sprintf(FindSaved, b.describeFilters(st.AgeMin, st.AgeMax, country)), nil)
}

func (b *Bot) describeFilters(minAge, maxAge int, country string) string {
	age := "any age"
	switch {
	case minAge > 0 && maxAge > 0 && minAge == maxAge:
		age = fmt.Sprintf("age %d", minAge)
	case minAge > 0 && maxAge > 0:
		age = fmt.Sprintf("age %d-%d", minAge, maxAge)
	case minAge > 0:
		age = fmt.Sprintf("age %d+", minAge)
	case maxAge > 0:
		age = fmt.Sprintf("up to age %d", maxAge)
	}
	place := "any country"
	if country != "" {
		place = country
	}
	return age + ", " + place
}
