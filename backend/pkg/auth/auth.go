package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	"golang.org/x/crypto/bcrypt"
)

const (
	BcryptCost       = 12
	TokenIssuer      = "panchakarma-identity-service"
	TokenAudience    = "panchakarma-platform"
	DefaultAccessTTL = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour
)

// Standard JWT custom claims
type Claims struct {
	UserID   string `json:"sub"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	ClinicID string `json:"clinic_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager handles generation, cryptographic validation, and blacklisting of JWTs and refresh tokens.
type TokenManager struct {
	secretKey      []byte
	accessTTL      time.Duration
	refreshTTL     time.Duration
	redisClient    *redis.Client
	localBlacklist map[string]time.Time
	mu             sync.RWMutex
}

// NewTokenManager initializes the TokenManager with secret and Redis/in-memory blacklist.
func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration, rdb *redis.Client) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret key must be at least 32 characters long")
	}
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}

	return &TokenManager{
		secretKey:      []byte(secret),
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
		redisClient:    rdb,
		localBlacklist: make(map[string]time.Time),
	}, nil
}

// GetAccessTTL returns the configured access token TTL.
func (m *TokenManager) GetAccessTTL() time.Duration {
	return m.accessTTL
}

// GetRefreshTTL returns the configured refresh token TTL.
func (m *TokenManager) GetRefreshTTL() time.Duration {
	return m.refreshTTL
}

// GenerateAccessToken generates a short-lived signed JWT with claims.
func (m *TokenManager) GenerateAccessToken(userID, email, role, clinicID string) (tokenStr string, jti string, expiresAt time.Time, err error) {
	now := time.Now().UTC()
	expiresAt = now.Add(m.accessTTL)
	jti = uuid.New().String()

	claims := Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		ClinicID: clinicID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err = token.SignedString(m.secretKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenStr, jti, expiresAt, nil
}

// VerifyAccessToken parses and cryptographically validates the token.
func (m *TokenManager) VerifyAccessToken(ctx context.Context, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		// Strict check against algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	}, jwt.WithIssuer(TokenIssuer), jwt.WithAudience(TokenAudience))

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if claims.ID == "" {
		return nil, errors.New("token is missing jti claim")
	}

	// Verify token is not blacklisted
	isBlacklisted, err := m.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("blacklist check failed: %w", err)
	}
	if isBlacklisted {
		return nil, errors.New("token has been revoked")
	}

	return claims, nil
}

// GenerateRefreshToken generates a 256-bit cryptographically secure opaque token and its SHA-256 hash.
func (m *TokenManager) GenerateRefreshToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random token bytes: %w", err)
	}

	rawToken = hex.EncodeToString(bytes)
	tokenHash = HashToken(rawToken)
	return rawToken, tokenHash, nil
}

// HashToken computes the SHA-256 hash of an opaque token.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// BlacklistToken adds a JWT ID (jti) to the revocation blacklist.
func (m *TokenManager) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}

	if m.redisClient != nil && m.redisClient.Client != nil {
		key := fmt.Sprintf("panchakarma:blacklist:jti:%s", jti)
		if err := m.redisClient.Set(ctx, key, "revoked", ttl).Err(); err == nil {
			return nil
		}
	}

	// Fallback to in-memory blacklist
	m.mu.Lock()
	defer m.mu.Unlock()
	m.localBlacklist[jti] = time.Now().Add(ttl)
	return nil
}

// IsBlacklisted checks if the JWT ID (jti) is in Redis or local memory blacklist.
func (m *TokenManager) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if m.redisClient != nil && m.redisClient.Client != nil {
		key := fmt.Sprintf("panchakarma:blacklist:jti:%s", jti)
		exists, err := m.redisClient.Exists(ctx, key).Result()
		if err == nil {
			return exists > 0, nil
		}
	}

	// Check local memory fallback
	m.mu.RLock()
	defer m.mu.RUnlock()
	expiry, found := m.localBlacklist[jti]
	if !found {
		return false, nil
	}
	if time.Now().After(expiry) {
		return false, nil
	}
	return true, nil
}

// HashPassword hashes a raw password using bcrypt with cost 12.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPasswordHash compares a plaintext password with its bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
