package game

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type User struct {
	Username  string
	IQ        int
	LastPlays []time.Time // Store recent play timestamps for rate limiting
}

type Manager struct {
	users      map[string]*User
	chatLimits map[int64][]time.Time // Track rate limits per chat
	mu         sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		users:      make(map[string]*User),
		chatLimits: make(map[int64][]time.Time),
	}
}

func (m *Manager) PlayGame(chatID int64, username string) (int, int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if !m.checkRateLimit(chatID, now) {
		return 0, 0, false
	}

	user, exists := m.users[username]
	if !exists {
		user = &User{
			Username:  username,
			IQ:        100, // Start with average IQ of 100
			LastPlays: make([]time.Time, 0),
		}
		m.users[username] = user
	}

	rand.Seed(time.Now().UnixNano())
	iqChange := rand.Intn(151) - 50 // Range from -50 to 100

	user.IQ += iqChange
	user.LastPlays = append(user.LastPlays, now)

	return iqChange, user.IQ, true
}

// Returns true if request is allowed, false if rate limited
func (m *Manager) checkRateLimit(chatID int64, now time.Time) bool {
	timestamps, exists := m.chatLimits[chatID]
	if !exists {
		timestamps = make([]time.Time, 0)
	}

	tenSecondsAgo := now.Add(-10 * time.Second)
	validTimestamps := make([]time.Time, 0)
	for _, ts := range timestamps {
		if ts.After(tenSecondsAgo) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	if len(validTimestamps) >= 10 {
		// Rate limit exceeded
		m.chatLimits[chatID] = validTimestamps
		return false
	}

	validTimestamps = append(validTimestamps, now)
	m.chatLimits[chatID] = validTimestamps
	return true
}

func (m *Manager) GetUserScore(username string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[username]
	if !exists {
		return 100 // Default IQ for new users
	}
	return user.IQ
}

func (m *Manager) GetTopUsers(limit int) []User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].IQ > users[j].IQ
	})

	if len(users) <= limit {
		return users
	}
	return users[:limit]
}

func (m *Manager) FormatLeaderboard() string {
	topUsers := m.GetTopUsers(10)

	if len(topUsers) == 0 {
		return "Пока никто не тестировал свой IQ. Будьте первым гением!"
	}

	leaderboard := ""

	// Titles based on position
	titles := []string{
		"🥇 Гений человечества",
		"🥈 Почти Эйнштейн",
		"🥉 Подающий надежды",
		"4️⃣ Смышлёный тип",
		"5️⃣ Начитанный эрудит",
	}

	for i, user := range topUsers {
		if i < len(titles) {
			leaderboard += fmt.Sprintf("%s: *%s* — %d IQ\n", titles[i], user.Username, user.IQ)
		} else {
			leaderboard += fmt.Sprintf("%d. *%s* — %d IQ\n", i+1, user.Username, user.IQ)
		}
	}

	return leaderboard
}
