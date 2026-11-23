package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"strangerbot/internal/config"
)

// User is one bot user and their profile / economy state.
type User struct {
	ID                int64    `json:"id" bson:"_id"`
	Username          string   `json:"username"`
	FirstName         string   `json:"first_name"`
	Age               int      `json:"age"`
	Gender            string   `json:"gender"`
	Country           string   `json:"country"`
	Interests         []string `json:"interests"`
	Registered        bool     `json:"registered"`
	SearchesUsedToday int      `json:"searches_used_today"`
	SearchesDate      string   `json:"searches_date"`
	BonusSearches     int      `json:"bonus_searches"`
	LastDaily         string   `json:"last_daily"`
	DailyStreak       int      `json:"daily_streak"`
	Premium           bool     `json:"premium"`
	Blocked           []int64  `json:"blocked"`
	Likes             int      `json:"likes"`
	Dislikes          int      `json:"dislikes"`
	TotalChats        int      `json:"total_chats"`
	Banned            bool     `json:"banned"`
	BanReason         string   `json:"ban_reason"`
	Joined            string   `json:"joined"`
	Bio               string   `json:"bio"`
	Lang              string   `json:"lang"`

	WantAgeMin    int    `json:"want_age_min" bson:"want_age_min"`
	WantAgeMax    int    `json:"want_age_max" bson:"want_age_max"`
	WantCountry   string `json:"want_country" bson:"want_country"`
	WantRegion    string `json:"want_region" bson:"want_region"`
	Privacy       bool   `json:"privacy" bson:"privacy"`
	ReferredBy    int64  `json:"referred_by" bson:"referred_by"`
	ReferralCount int    `json:"referral_count" bson:"referral_count"`
	PremiumUntil  string `json:"premium_until" bson:"premium_until"`
	TotalReports  int    `json:"total_reports" bson:"total_reports"`

	Verified   bool   `json:"verified" bson:"verified"`
	Strikes    int    `json:"strikes" bson:"strikes"`
	MutedUntil string `json:"muted_until" bson:"muted_until"`
}

// Waiting is a user looking for a partner.
type Waiting struct {
	ID   int64  `json:"id" bson:"_id"`
	Want string `json:"want"`
}

// Report is a user-submitted abuse report.
type Report struct {
	ID       int64  `json:"id" bson:"_id"`
	Reporter int64  `json:"reporter"`
	Reported int64  `json:"reported"`
	Reason   string `json:"reason"`
	Time     string `json:"time"`
	Resolved bool   `json:"resolved"`
}

// Feedback is a user message sent to the admins.
type Feedback struct {
	ID     int64  `json:"id" bson:"_id"`
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

type data struct {
	Users    map[string]*User  `json:"users"`
	Waiting  []Waiting         `json:"waiting"`
	Pairs    map[string]int64  `json:"pairs"`
	Reports  []Report          `json:"reports"`
	Feedback []Feedback        `json:"feedback"`
	Meta     map[string]string `json:"meta"`
}

// Stats is an aggregated snapshot for admin /stats and /stats public view.
type Stats struct {
	TotalUsers    int `json:"total_users"`
	Registered    int `json:"registered"`
	ActiveChats   int `json:"active_chats"`
	Waiting       int `json:"waiting"`
	Premium       int `json:"premium"`
	Banned        int `json:"banned"`
	OpenReports   int `json:"open_reports"`
	TotalReports  int `json:"total_reports"`
	TotalMessages int `json:"total_messages"`
}

// Store is a JSON-file backed database with atomic writes.
type Store struct {
	mu   sync.Mutex
	path string
	cfg  *config.Config
	data data
	mb   *mongoBackend
}

// Open loads (or creates) the database. When DB_DRIVER is mongodb the state
// lives in MongoDB, otherwise it is kept in the JSON file at path.
func Open(path string, cfg *config.Config) (*Store, error) {
	s := &Store{path: path, cfg: cfg, data: newData()}

	if mongoDriverName(strings.ToLower(strings.TrimSpace(cfg.DBDriver))) {
		mb, err := openMongo(cfg.MongoURI, cfg.MongoDatabase)
		if err != nil {
			return nil, fmt.Errorf("connect mongodb: %w", err)
		}
		d, err := mb.load()
		if err != nil {
			mb.close()
			return nil, fmt.Errorf("load mongodb: %w", err)
		}
		s.mb = mb
		s.data = normalize(d)
		return s, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	var d data
	if err := json.Unmarshal(raw, &d); err != nil {
		// Never clobber a corrupt file: move it aside and start clean.
		_ = os.Rename(path, path+".corrupt")
		return s, nil
	}
	s.data = normalize(d)
	return s, nil
}

func newData() data {
	return data{
		Users: map[string]*User{},
		Pairs: map[string]int64{},
		Meta:  map[string]string{},
	}
}

func normalize(d data) data {
	if d.Users == nil {
		d.Users = map[string]*User{}
	}
	if d.Pairs == nil {
		d.Pairs = map[string]int64{}
	}
	if d.Meta == nil {
		d.Meta = map[string]string{}
	}
	if d.Waiting == nil {
		d.Waiting = []Waiting{}
	}
	if d.Reports == nil {
		d.Reports = []Report{}
	}
	if d.Feedback == nil {
		d.Feedback = []Feedback{}
	}
	seen := map[int64]bool{}
	clean := d.Waiting[:0]
	for _, w := range d.Waiting {
		if w.ID == 0 || seen[w.ID] {
			continue
		}
		if w.Want != "male" && w.Want != "female" {
			w.Want = ""
		}
		seen[w.ID] = true
		clean = append(clean, w)
	}
	d.Waiting = clean
	// Drop pair references to unknown users.
	for uid, partner := range d.Pairs {
		if _, ok := d.Users[uid]; !ok {
			delete(d.Pairs, uid)
			continue
		}
		if _, ok := d.Users[fmt.Sprint(partner)]; !ok {
			delete(d.Pairs, uid)
		}
	}
	return d
}

func today() string { return time.Now().UTC().Format("2006-01-02") }

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func key(uid int64) string { return fmt.Sprint(uid) }

// saveLocked writes the in-memory state atomically. Caller must hold s.mu.
func (s *Store) saveLocked() {
	tmp := s.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s.data); err != nil {
		f.Close()
		return
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return
	}
	if err := f.Close(); err != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}

// persistUser writes one user through the active backend.
func (s *Store) persistUser(u *User) {
	if s.mb != nil {
		s.mb.saveUser(u)
		return
	}
	s.saveLocked()
}

// persistState writes the shared state through the active backend.
func (s *Store) persistState() {
	if s.mb != nil {
		s.mb.saveState(&s.data)
		return
	}
	s.saveLocked()
}

func defaultUser(uid int64, uname, fname string) *User {
	return &User{
		ID:           uid,
		Username:     uname,
		FirstName:    fname,
		Interests:    []string{},
		Blocked:      []int64{},
		SearchesDate: today(),
		Joined:       now(),
	}
}

func available(u *User, cfg *config.Config) int {
	daily := cfg.FreeDailySearches
	if u.Premium {
		daily = cfg.PremiumDailySearches
	}
	left := daily - u.SearchesUsedToday + u.BonusSearches
	if left < 0 {
		return 0
	}
	return left
}

func (s *Store) resetDailyLocked(u *User) bool {
	if u.SearchesDate != today() {
		u.SearchesUsedToday = 0
		u.SearchesDate = today()
		return true
	}
	return false
}

// EnsureUser creates the user if missing and refreshes profile fields.
func (s *Store) EnsureUser(uid int64, uname, fname string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.data.Users[key(uid)]
	if u == nil {
		u = defaultUser(uid, uname, fname)
		s.data.Users[key(uid)] = u
		s.persistUser(u)
	} else if (uname != "" && u.Username != uname) || (fname != "" && u.FirstName != fname) {
		if uname != "" {
			u.Username = uname
		}
		if fname != "" {
			u.FirstName = fname
		}
		s.persistUser(u)
	}
	return *u
}

// GetUser returns a copy of the user, if present.
func (s *Store) GetUser(uid int64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return User{}, false
	}
	return *u, true
}

// UpdateUser applies fn to the stored user and persists the result.
func (s *Store) UpdateUser(uid int64, fn func(*User)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	fn(u)
	s.persistUser(u)
}

// AllUsers returns copies of every user.
func (s *Store) AllUsers() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]User, 0, len(s.data.Users))
	for _, u := range s.data.Users {
		out = append(out, *u)
	}
	return out
}

// SearchesLeft reports how many searches the user may still start today.
func (s *Store) SearchesLeft(uid int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return 0
	}
	if s.resetDailyLocked(u) {
		s.persistUser(u)
	}
	return available(u, s.cfg)
}

// ConsumeSearch spends one search (bonus first) and reports success.
func (s *Store) ConsumeSearch(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return false
	}
	s.resetDailyLocked(u)
	if available(u, s.cfg) <= 0 {
		s.persistUser(u)
		return false
	}
	if u.BonusSearches > 0 {
		u.BonusSearches--
	} else {
		u.SearchesUsedToday++
	}
	s.persistUser(u)
	return true
}

// AddBonusSearches adjusts the bonus pool (amount may be negative).
func (s *Store) AddBonusSearches(uid int64, amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return
	}
	u.BonusSearches += amount
	if u.BonusSearches < 0 {
		u.BonusSearches = 0
	}
	s.persistUser(u)
}

// ClaimDaily grants the daily bonus once per day.
func (s *Store) ClaimDaily(uid int64) (amount, streak int, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok2 := s.data.Users[key(uid)]
	if !ok2 {
		return 0, 0, false
	}
	t := today()
	if u.LastDaily == t {
		return 0, u.DailyStreak, false
	}
	streak = 1
	if u.LastDaily != "" {
		if y, err := time.Parse("2006-01-02", u.LastDaily); err == nil {
			if y.AddDate(0, 0, 1).Format("2006-01-02") == t {
				streak = u.DailyStreak + 1
			}
		}
	}
	amount = s.cfg.DailyBonusBase + s.cfg.DailyBonusStep*(streak-1)
	if amount > s.cfg.DailyBonusMax {
		amount = s.cfg.DailyBonusMax
	}
	u.LastDaily = t
	u.DailyStreak = streak
	u.BonusSearches += amount
	s.persistUser(u)
	return amount, streak, true
}

// GetInterests returns the user's selected interests.
func (s *Store) GetInterests(uid int64) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return nil
	}
	return append([]string(nil), u.Interests...)
}

// ToggleInterest adds/removes an interest. Returns added, removed, limit, invalid.
func (s *Store) ToggleInterest(uid int64, interest string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !config.Valid(interest) {
		return "invalid"
	}
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return "invalid"
	}
	for i, v := range u.Interests {
		if v == interest {
			u.Interests = append(u.Interests[:i], u.Interests[i+1:]...)
			s.persistUser(u)
			return "removed"
		}
	}
	if len(u.Interests) >= s.cfg.MaxInterests {
		return "limit"
	}
	u.Interests = append(u.Interests, interest)
	s.persistUser(u)
	return "added"
}

// ClearInterests empties the interest list.
func (s *Store) ClearInterests(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		u.Interests = []string{}
		s.persistUser(u)
	}
}

// AddWaiting puts a user in the queue (idempotent).
func (s *Store) AddWaiting(uid int64, want string) {
	if want != "male" && want != "female" {
		want = ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.data.Waiting {
		if w.ID == uid {
			return
		}
	}
	s.data.Waiting = append(s.data.Waiting, Waiting{ID: uid, Want: want})
	s.persistState()
}

// RemoveWaiting removes a user from the queue.
func (s *Store) RemoveWaiting(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.data.Waiting)
	out := s.data.Waiting[:0]
	for _, w := range s.data.Waiting {
		if w.ID != uid {
			out = append(out, w)
		}
	}
	if len(out) != before {
		s.data.Waiting = out
		s.persistState()
	}
}

// GetWaiting returns a copy of the queue.
func (s *Store) GetWaiting() []Waiting {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Waiting(nil), s.data.Waiting...)
}

// IsWaiting reports whether the user is queued.
func (s *Store) IsWaiting(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.data.Waiting {
		if w.ID == uid {
			return true
		}
	}
	return false
}

// TryPair links two waiting users. It fails unless both are queued and free.
func (s *Store) TryPair(a, b int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	qa, qb := false, false
	for _, w := range s.data.Waiting {
		if w.ID == a {
			qa = true
		}
		if w.ID == b {
			qb = true
		}
	}
	if !qa || !qb {
		return false
	}
	if _, ok := s.data.Pairs[key(a)]; ok {
		return false
	}
	if _, ok := s.data.Pairs[key(b)]; ok {
		return false
	}
	waiting := s.data.Waiting[:0]
	for _, w := range s.data.Waiting {
		if w.ID != a && w.ID != b {
			waiting = append(waiting, w)
		}
	}
	s.data.Waiting = waiting
	s.data.Pairs[key(a)] = b
	s.data.Pairs[key(b)] = a
	for _, uid := range []int64{a, b} {
		if u, ok := s.data.Users[key(uid)]; ok {
			u.TotalChats++
		}
	}
	s.persistState()
	return true
}

// GetPartner returns the current partner id.
func (s *Store) GetPartner(uid int64) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Pairs[key(uid)]
	return p, ok
}

// RemovePair unlinks a user and their partner, returning the partner id.
func (s *Store) RemovePair(uid int64) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.data.Pairs[key(uid)]
	if !ok {
		return 0, false
	}
	delete(s.data.Pairs, key(uid))
	delete(s.data.Pairs, key(p))
	s.persistState()
	return p, true
}

// IsInChat reports whether the user is paired.
func (s *Store) IsInChat(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data.Pairs[key(uid)]
	return ok
}

// BlockUser blocks target for blocker. Returns self, already, limit or ok.
func (s *Store) BlockUser(blocker, target int64) string {
	if blocker == target {
		return "self"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(blocker)]
	if !ok {
		return "self"
	}
	for _, b := range u.Blocked {
		if b == target {
			return "already"
		}
	}
	if len(u.Blocked) >= s.cfg.MaxBlockList {
		return "limit"
	}
	u.Blocked = append(u.Blocked, target)
	s.persistUser(u)
	return "ok"
}

// UnblockByIndex removes the 1-based entry from the block list.
func (s *Store) UnblockByIndex(blocker int64, idx int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(blocker)]
	if !ok {
		return "not_found"
	}
	if idx < 1 || idx > len(u.Blocked) {
		return "not_found"
	}
	u.Blocked = append(u.Blocked[:idx-1], u.Blocked[idx:]...)
	s.persistUser(u)
	return "ok"
}

// GetBlocklist returns the user's blocked ids.
func (s *Store) GetBlocklist(uid int64) []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	if !ok {
		return nil
	}
	return append([]int64(nil), u.Blocked...)
}

// IsBlockedBy reports whether blocker has blocked target.
func (s *Store) IsBlockedBy(blocker, target int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(blocker)]
	if !ok {
		return false
	}
	for _, b := range u.Blocked {
		if b == target {
			return true
		}
	}
	return false
}

// ApplyLike rewards the target with bonus searches.
func (s *Store) ApplyLike(_, target int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(target)]; ok {
		u.BonusSearches += s.cfg.LikeReward
		u.Likes++
		s.persistUser(u)
	}
}

// ApplyDislike costs the target a bonus search.
func (s *Store) ApplyDislike(_, target int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(target)]; ok {
		u.BonusSearches -= s.cfg.DislikeCost
		if u.BonusSearches < 0 {
			u.BonusSearches = 0
		}
		u.Dislikes++
		s.persistUser(u)
	}
}

// AddReport stores a report and returns it with its assigned id.
func (s *Store) AddReport(reporter, reported int64, reason string) Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := int64(0)
	for _, r := range s.data.Reports {
		if r.ID > id {
			id = r.ID
		}
	}
	id++
	if u, ok := s.data.Users[key(reported)]; ok {
		u.TotalReports++
		s.persistUser(u)
	}
	r := Report{ID: id, Reporter: reporter, Reported: reported, Reason: reason, Time: now()}
	s.data.Reports = append(s.data.Reports, r)
	s.persistState()
	return r
}

// GetReports returns reports filtered by resolved state.
func (s *Store) GetReports(resolved bool) []Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Report, 0)
	for _, r := range s.data.Reports {
		if r.Resolved == resolved {
			out = append(out, r)
		}
	}
	return out
}

// ResolveReport marks a report resolved.
func (s *Store) ResolveReport(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Reports {
		if s.data.Reports[i].ID == id {
			s.data.Reports[i].Resolved = true
			s.persistState()
			return true
		}
	}
	return false
}

// BanUser / UnbanUser / IsBanned manage bans.
func (s *Store) BanUser(uid int64, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		u.Banned = true
		u.BanReason = reason
		s.persistUser(u)
	}
}

func (s *Store) UnbanUser(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.data.Users[key(uid)]; ok {
		u.Banned = false
		u.BanReason = ""
		s.persistUser(u)
	}
}

func (s *Store) IsBanned(uid int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[key(uid)]
	return ok && u.Banned
}

// AddFeedback stores feedback for admins.
func (s *Store) AddFeedback(uid int64, text string) Feedback {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := int64(0)
	for _, f := range s.data.Feedback {
		if f.ID > id {
			id = f.ID
		}
	}
	id++
	f := Feedback{ID: id, UserID: uid, Text: text, Time: now()}
	s.data.Feedback = append(s.data.Feedback, f)
	s.persistState()
	return f
}

// GetFeedback returns all feedback.
func (s *Store) GetFeedback() []Feedback {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Feedback(nil), s.data.Feedback...)
}

// SetMeta stores an arbitrary string setting.
func (s *Store) SetMeta(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Meta[k] = v
	s.persistState()
}

// GetMeta reads an arbitrary string setting.
func (s *Store) GetMeta(k string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data.Meta[k]
	return v, ok
}

// Stats aggregates counters.
func (s *Store) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	var st Stats
	for _, u := range s.data.Users {
		st.TotalUsers++
		if u.Registered {
			st.Registered++
		}
		if u.Premium {
			st.Premium++
		}
		if u.Banned {
			st.Banned++
		}
	}
	st.ActiveChats = len(s.data.Pairs) / 2
	st.Waiting = len(s.data.Waiting)
	st.OpenReports = 0
	for _, r := range s.data.Reports {
		if !r.Resolved {
			st.OpenReports++
		}
	}
	st.TotalReports = len(s.data.Reports)
	return st
}

// TopChatters returns the most active registered users.
func (s *Store) TopChatters(limit int) []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []User
	for _, u := range s.data.Users {
		if u.Registered {
			list = append(list, *u)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TotalChats != list[j].TotalChats {
			return list[i].TotalChats > list[j].TotalChats
		}
		if list[i].Likes != list[j].Likes {
			return list[i].Likes > list[j].Likes
		}
		return list[i].ID < list[j].ID
	})
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list
}

// Rank returns the user's 1-based rank by chats and the total ranked users.
func (s *Store) Rank(uid int64) (rank, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []User
	for _, u := range s.data.Users {
		if u.Registered {
			list = append(list, *u)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TotalChats != list[j].TotalChats {
			return list[i].TotalChats > list[j].TotalChats
		}
		if list[i].Likes != list[j].Likes {
			return list[i].Likes > list[j].Likes
		}
		return list[i].ID < list[j].ID
	})
	total = len(list)
	for i, u := range list {
		if u.ID == uid {
			return i + 1, total
		}
	}
	return 0, total
}

// CountOpenReportsAgainst counts unresolved reports for a user.
func (s *Store) CountOpenReportsAgainst(uid int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.data.Reports {
		if r.Reported == uid && !r.Resolved {
			n++
		}
	}
	return n
}

// Close is a no-op kept for interface symmetry.
func (s *Store) Close() error {
	if s.mb != nil {
		s.mb.close()
		s.mb = nil
	}
	return nil
}
