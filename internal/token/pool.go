package token

import (
	"math/rand/v2"
	"sync"
)

type TokenPool struct {
	name   string
	tokens map[string]*TokenInfo
	mu     sync.RWMutex
}

func NewTokenPool(name string) *TokenPool {
	return &TokenPool{
		name:   name,
		tokens: make(map[string]*TokenInfo),
	}
}

func (p *TokenPool) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

func (p *TokenPool) Add(token *TokenInfo) {
	if p == nil || token == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokens[NormalizeToken(token.Token)] = token
}

func (p *TokenPool) Remove(tokenStr string) bool {
	if p == nil {
		return false
	}
	tokenKey := NormalizeToken(tokenStr)
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.tokens[tokenKey]; !ok {
		return false
	}
	delete(p.tokens, tokenKey)
	return true
}

func (p *TokenPool) Get(tokenStr string) *TokenInfo {
	if p == nil {
		return nil
	}
	tokenKey := NormalizeToken(tokenStr)
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tokens[tokenKey]
}

// Select picks an available token using least-used strategy.
// Among tokens with the lowest use count, one is chosen at random.
func (p *TokenPool) Select(exclude map[string]bool) *TokenInfo {
	if p == nil {
		return nil
	}

	p.mu.RLock()
	available := make([]*TokenInfo, 0, len(p.tokens))
	for _, token := range p.tokens {
		if exclude != nil && exclude[NormalizeToken(token.Token)] {
			continue
		}
		if token.IsAvailable() {
			available = append(available, token)
		}
	}
	p.mu.RUnlock()

	if len(available) == 0 {
		return nil
	}

	// Find minimum use count
	minUse := available[0].UseCount
	for _, t := range available[1:] {
		if t.UseCount < minUse {
			minUse = t.UseCount
		}
	}

	// Collect all tokens with minimum use count
	candidates := make([]*TokenInfo, 0, len(available))
	for _, t := range available {
		if t.UseCount == minUse {
			candidates = append(candidates, t)
		}
	}

	return candidates[rand.IntN(len(candidates))]
}

func (p *TokenPool) Count() int {
	if p == nil {
		return 0
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.tokens)
}

func (p *TokenPool) List() []*TokenInfo {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	items := make([]*TokenInfo, 0, len(p.tokens))
	for _, token := range p.tokens {
		items = append(items, token)
	}
	return items
}

func (p *TokenPool) GetStats() TokenPoolStats {
	stats := TokenPoolStats{}
	if p == nil {
		return stats
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	stats.Total = len(p.tokens)
	for _, token := range p.tokens {
		switch token.Status {
		case StatusActive:
			stats.Active++
		case StatusDisabled:
			stats.Disabled++
		case StatusExpired:
			stats.Expired++
		case StatusCooling:
			stats.Cooling++
		}
	}
	return stats
}
