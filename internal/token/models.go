package token

import (
	"strings"
	"time"
)

const (
	FailThreshold = 5
)

type TokenStatus string

const (
	StatusActive   TokenStatus = "active"
	StatusDisabled TokenStatus = "disabled"
	StatusExpired  TokenStatus = "expired"
	StatusCooling  TokenStatus = "cooling"
)

type TokenInfo struct {
	Token          string      `json:"token"`
	Status         TokenStatus `json:"status"`
	CreatedAt      int64       `json:"created_at"`
	LastUsedAt     int64       `json:"last_used_at,omitempty"`
	UseCount       int         `json:"use_count"`
	FailCount      int         `json:"fail_count"`
	LastFailAt     int64       `json:"last_fail_at,omitempty"`
	LastFailReason string      `json:"last_fail_reason,omitempty"`
	Note           string      `json:"note,omitempty"`
}

type TokenPoolStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Disabled int `json:"disabled"`
	Expired  int `json:"expired"`
	Cooling  int `json:"cooling"`
}

func NewTokenInfo(token string) *TokenInfo {
	return &TokenInfo{
		Token:     NormalizeToken(token),
		Status:    StatusActive,
		CreatedAt: nowMillis(),
	}
}

// NormalizeToken trims whitespace. ChatGPT tokens are JWTs, not SSO cookies.
func NormalizeToken(value string) string {
	return strings.TrimSpace(value)
}

func (t *TokenInfo) IsAvailable() bool {
	return t != nil && t.Status == StatusActive
}

func (t *TokenInfo) RecordFail(reason string, threshold int) {
	if t == nil {
		return
	}
	if threshold <= 0 {
		threshold = FailThreshold
	}
	t.FailCount++
	t.LastFailAt = nowMillis()
	t.LastFailReason = reason
	if t.FailCount >= threshold {
		t.EnterCooling()
	}
}

func (t *TokenInfo) RecordSuccess() {
	if t == nil {
		return
	}
	t.FailCount = 0
	t.LastFailAt = 0
	t.LastFailReason = ""
	t.UseCount++
	t.LastUsedAt = nowMillis()
}

func (t *TokenInfo) EnterCooling() {
	if t == nil {
		return
	}
	t.Status = StatusCooling
}

func (t *TokenInfo) RecoverActive() {
	if t == nil {
		return
	}
	if t.Status == StatusCooling {
		t.Status = StatusActive
		t.FailCount = 0
		t.LastFailAt = 0
		t.LastFailReason = ""
	}
}

func (t *TokenInfo) Clone() *TokenInfo {
	if t == nil {
		return nil
	}
	clone := *t
	return &clone
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}
