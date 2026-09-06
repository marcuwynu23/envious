package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"envious-web/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

// InitAdminKey ensures a bcrypt-hashed API key exists. If none exists, it generates
// a new random API key, stores the hash, and returns the plaintext key for one-time display.
func InitAdminKey(ctx context.Context, s *storage.Storage) (string, error) {
	_, err := s.GetAPIKeyHash(ctx)
	if err == nil {
		return "", nil
	}
	if err != nil && err != storage.ErrNotFound {
		return "", err
	}
	key, err := generateAPIKey()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if err := s.SetAPIKeyHash(ctx, string(hash)); err != nil {
		return "", err
	}
	return key, nil
}

func Verify(ctx context.Context, s *storage.Storage, provided string) bool {
	hash, err := s.GetAPIKeyHash(ctx)
	if err != nil {
		log.Printf("auth: could not load hash: %v", err)
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(provided)) == nil
}

// CachedVerifier holds the API-key bcrypt hash in memory so high-traffic
// deployments don't pay a DB read plus ~100ms of bcrypt CPU per request.
// The hash reloads after ttl (key rotation takes effect on reload).
// A ttl <= 0 disables the cache (every call hits the database).
type CachedVerifier struct {
	mu   sync.Mutex
	hash string
	at   time.Time
	ttl  time.Duration
}

// NewCachedVerifier returns a verifier with the given cache TTL.
func NewCachedVerifier(ttl time.Duration) *CachedVerifier {
	return &CachedVerifier{ttl: ttl}
}

func (v *CachedVerifier) Verify(ctx context.Context, s *storage.Storage, provided string) bool {
	hash := v.cachedHash(ctx, s)
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(provided)) == nil
}

func (v *CachedVerifier) cachedHash(ctx context.Context, s *storage.Storage) string {
	if v.ttl > 0 {
		v.mu.Lock()
		hash, at := v.hash, v.at
		v.mu.Unlock()
		if hash != "" && time.Since(at) < v.ttl {
			return hash
		}
	}
	hash, err := s.GetAPIKeyHash(ctx)
	if err != nil {
		log.Printf("auth: could not load hash: %v", err)
		return ""
	}
	if v.ttl > 0 {
		v.mu.Lock()
		v.hash, v.at = hash, time.Now()
		v.mu.Unlock()
	}
	return hash
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

