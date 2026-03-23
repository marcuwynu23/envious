package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"

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

func generateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

