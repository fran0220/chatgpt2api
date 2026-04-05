package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"chatgpt2api/internal/storage"
)

const (
	DefaultReloadIntervalSec = 30
	DefaultPoolName          = "default"
)

type TokenManager struct {
	pool         *TokenPool
	storage      storage.Storage
	mu           sync.RWMutex
	lastReloadAt time.Time
}

var (
	managerOnce sync.Once
	managerInst *TokenManager
)

// GetInstance returns the singleton TokenManager. On first call it attempts
// to load tokens from storage; failures are logged but non-fatal.
func GetInstance() *TokenManager {
	managerOnce.Do(func() {
		managerInst = &TokenManager{
			pool: NewTokenPool(DefaultPoolName),
		}
	})
	return managerInst
}

// SetStorage sets the storage backend. Must be called before Load.
func (m *TokenManager) SetStorage(s storage.Storage) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage = s
}

func (m *TokenManager) Load() error {
	if m == nil {
		return errors.New("token manager is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.storage == nil {
		return errors.New("storage not configured")
	}

	loaded, err := m.storage.LoadTokens()
	if err != nil {
		return err
	}

	pool := NewTokenPool(DefaultPoolName)
	items := loaded[DefaultPoolName]
	for _, item := range items {
		info, err := tokenInfoFromData(item)
		if err != nil {
			slog.Warn("skip invalid token while loading", "error", err)
			continue
		}
		pool.Add(info)
	}

	m.pool = pool
	m.lastReloadAt = time.Now()
	return nil
}

func (m *TokenManager) Save() error {
	if m == nil {
		return errors.New("token manager is nil")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.storage == nil {
		return errors.New("storage not configured")
	}

	serialized, err := m.serializeLocked()
	if err != nil {
		return err
	}
	return m.storage.SaveTokens(serialized)
}

func (m *TokenManager) ReloadIfStale() {
	if m == nil {
		return
	}
	m.mu.RLock()
	stale := m.lastReloadAt.IsZero() || time.Since(m.lastReloadAt) >= DefaultReloadIntervalSec*time.Second
	m.mu.RUnlock()
	if !stale {
		return
	}
	if err := m.Load(); err != nil {
		slog.Warn("reload stale token pool failed", "error", err)
	}
}

// GetToken selects an available token string, excluding specified ones.
func (m *TokenManager) GetToken(exclude map[string]bool) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	pool := m.pool
	m.mu.RUnlock()

	selected := pool.Select(exclude)
	if selected == nil {
		return ""
	}
	return NormalizeToken(selected.Token)
}

func (m *TokenManager) GetTokenInfo(tokenStr string) *TokenInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pool.Get(tokenStr)
}

func (m *TokenManager) RecordFail(token, reason string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	info := m.pool.Get(token)
	if info == nil {
		return
	}
	info.RecordFail(reason, FailThreshold)
}

func (m *TokenManager) RecordSuccess(token string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	info := m.pool.Get(token)
	if info == nil {
		return
	}
	info.RecordSuccess()
}

func (m *TokenManager) AddToken(token, note string) error {
	if m == nil {
		return errors.New("token manager is nil")
	}
	normalized := NormalizeToken(token)
	if normalized == "" {
		return errors.New("token cannot be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.pool.Get(normalized); existing != nil {
		return fmt.Errorf("token already exists")
	}
	info := NewTokenInfo(normalized)
	info.Note = note
	m.pool.Add(info)
	return nil
}

func (m *TokenManager) RemoveToken(token string) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pool.Remove(token)
}

func (m *TokenManager) ListTokens() []*TokenInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pool.List()
}

func (m *TokenManager) GetStats() TokenPoolStats {
	if m == nil {
		return TokenPoolStats{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pool.GetStats()
}

func (m *TokenManager) serializeLocked() (map[string][]storage.TokenData, error) {
	items := m.pool.List()
	serialized := make([]storage.TokenData, 0, len(items))
	for _, item := range items {
		payload, err := tokenDataFromInfo(item)
		if err != nil {
			return nil, err
		}
		serialized = append(serialized, payload)
	}
	return map[string][]storage.TokenData{
		DefaultPoolName: serialized,
	}, nil
}

func tokenInfoFromData(data storage.TokenData) (*TokenInfo, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var info TokenInfo
	if err := json.Unmarshal(payload, &info); err != nil {
		return nil, err
	}
	info.Token = NormalizeToken(info.Token)
	if info.Token == "" {
		return nil, errors.New("token cannot be empty")
	}
	if info.Status == "" {
		info.Status = StatusActive
	}
	if info.CreatedAt == 0 {
		info.CreatedAt = nowMillis()
	}
	return &info, nil
}

func tokenDataFromInfo(info *TokenInfo) (storage.TokenData, error) {
	payload, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	var data storage.TokenData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	data["token"] = NormalizeToken(info.Token)
	return data, nil
}
