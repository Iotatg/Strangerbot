package store

import (
	"strconv"
	"time"
)

// SetFilters stores the search preferences of a user.
func (s *Store) SetFilters(uid int64, minAge, maxAge int, country string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.WantAgeMin = minAge
	u.WantAgeMax = maxAge
	u.WantCountry = country
	s.persistUser(u)
}

// SetPrivacy toggles auto-deletion of relayed messages.
func (s *Store) SetPrivacy(uid int64, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.Privacy = on
	s.persistUser(u)
}

// DeleteUser removes a user, their waiting entry and their pair.
func (s *Store) DeleteUser(uid int64) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Users[key(uid)]; !ok {
		return 0, false
	}
	partner, _ := s.data.Pairs[key(uid)]
	delete(s.data.Users, key(uid))
	delete(s.data.Pairs, key(uid))
	if partner != 0 {
		delete(s.data.Pairs, key(partner))
	}
	clean := s.data.Waiting[:0]
	for _, w := range s.data.Waiting {
		if w.ID != uid {
			clean = append(clean, w)
		}
	}
	s.data.Waiting = clean
	if s.mb != nil {
		s.mb.deleteUser(uid)
	}
	s.persistState()
	return partner, true
}

// ClearReports removes every report and returns how many were dropped.
func (s *Store) ClearReports() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.data.Reports)
	s.data.Reports = nil
	s.persistState()
	return n
}

// AllPremium returns a copy of every premium user.
func (s *Store) AllPremium() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]User, 0)
	for _, u := range s.data.Users {
		if u.Premium {
			out = append(out, *u)
		}
	}
	return out
}

// SetPremium grants or revokes premium with an optional expiry timestamp.
func (s *Store) SetPremium(uid int64, premium bool, until string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.Premium = premium
	u.PremiumUntil = until
	s.persistUser(u)
}

// CountReferrals returns how many users were referred by uid.
func (s *Store) CountReferrals(uid int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, u := range s.data.Users {
		if u.ReferredBy == uid {
			n++
		}
	}
	return n
}

// AttachReferral links a new user to their referrer and returns the referrer id.
func (s *Store) AttachReferral(uid, referrer int64) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok || u.ReferredBy != 0 || referrer == uid {
		return 0, false
	}
	ref, ok := s.data.Users[key(referrer)]
	if !ok || ref.Banned {
		return 0, false
	}
	u.ReferredBy = referrer
	ref.ReferralCount++
	s.persistUser(u)
	s.persistUser(ref)
	return referrer, true
}

// ExpirePremium revokes premium whose paid period has elapsed.
func (s *Store) ExpirePremium(now time.Time) []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []int64
	for _, u := range s.data.Users {
		if !u.Premium || u.PremiumUntil == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, u.PremiumUntil)
		if err != nil || now.After(t) {
			u.Premium = false
			u.PremiumUntil = ""
			expired = append(expired, u.ID)
			s.persistUser(u)
		}
	}
	return expired
}

// SetRegion stores the preferred match region (india, foreign or "" for any).
func (s *Store) SetRegion(uid int64, region string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.WantRegion = region
	s.persistUser(u)
}

// GetRegion returns the stored match region preference.
func (s *Store) GetRegion(uid int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		return u.WantRegion
	}
	return ""
}

// GetFilters returns the stored search preferences.
func (s *Store) GetFilters(uid int64) (minAge, maxAge int, country string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		return u.WantAgeMin, u.WantAgeMax, u.WantCountry
	}
	return 0, 0, ""
}

// AllPairs returns a snapshot of every active pair.
func (s *Store) AllPairs() [][2]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][2]int64, 0, len(s.data.Pairs))
	seen := map[int64]bool{}
	for uidStr, partner := range s.data.Pairs {
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil {
			continue
		}
		if seen[uid] || seen[partner] {
			continue
		}
		seen[uid] = true
		seen[partner] = true
		out = append(out, [2]int64{uid, partner})
	}
	return out
}

// TopTrust returns users ordered by a trust score, best first.
func (s *Store) TopTrust(limit int) []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]User, 0, len(s.data.Users))
	for _, u := range s.data.Users {
		if !u.Banned {
			users = append(users, *u)
		}
	}
	sortUsersByTrust(users)
	if len(users) > limit {
		users = users[:limit]
	}
	return users
}

// TrustScore is a 0-100 reputation value derived from activity, ratings and
// moderation history.
func TrustScore(u User) int {
	score := 60 + u.TotalChats + u.Likes*2 - u.Dislikes*3 - u.TotalReports*10
	if u.Banned {
		score -= 40
	}
	if u.Premium {
		score += 5
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func sortUsersByTrust(users []User) {
	for i := 1; i < len(users); i++ {
		for j := i; j > 0 && TrustScore(users[j]) > TrustScore(users[j-1]); j-- {
			users[j], users[j-1] = users[j-1], users[j]
		}
	}
}

// SetVerified marks a user as having passed the human check.
func (s *Store) SetVerified(uid int64, verified bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.Verified = verified
	s.persistUser(u)
}

// IsVerified reports whether the user passed the human check.
func (s *Store) IsVerified(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	return ok && u.Verified
}

// AddStrike records one violation and returns the new total.
func (s *Store) AddStrike(uid int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return 0
	}
	u.Strikes++
	n := u.Strikes
	s.persistUser(u)
	return n
}

// ResetStrikes clears the violation counter.
func (s *Store) ResetStrikes(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		u.Strikes = 0
		s.persistUser(u)
	}
}

// SetMuted mutes a user until the given RFC3339 timestamp.
func (s *Store) SetMuted(uid int64, until string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		u.MutedUntil = until
		s.persistUser(u)
	}
}

// IsMuted reports whether the user is currently muted.
func (s *Store) IsMuted(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok || u.MutedUntil == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, u.MutedUntil)
	if err != nil {
		return false
	}
	return time.Now().UTC().Before(t)
}

// OpenReportsAgainst counts unresolved reports for a user.
func (s *Store) OpenReportsAgainst(uid int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.data.Reports {
		if !r.Resolved && r.Reported == uid {
			n++
		}
	}
	return n
}
