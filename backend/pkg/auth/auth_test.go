package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
)

const testSecret = "panchakarma_flow_super_secret_test_key_at_least_32_bytes_long"

func TestTokenManager_Lifecycle(t *testing.T) {
	tm, err := auth.NewTokenManager(testSecret, 2*time.Second, 1*time.Hour, nil)
	if err != nil {
		t.Fatalf("failed to create TokenManager: %v", err)
	}

	userID := uuid.New().String()
	email := "test.patient@example.com"
	role := "patient"
	clinicID := ""

	t.Run("ValidAccessTokenGenerationAndVerification", func(t *testing.T) {
		tokenStr, jti, exp, err := tm.GenerateAccessToken(userID, email, role, clinicID)
		if err != nil {
			t.Fatalf("unexpected error generating token: %v", err)
		}
		if tokenStr == "" || jti == "" || exp.IsZero() {
			t.Fatalf("expected non-empty token string, jti and expiration")
		}

		claims, err := tm.VerifyAccessToken(context.Background(), tokenStr)
		if err != nil {
			t.Fatalf("failed to verify valid token: %v", err)
		}
		if claims.UserID != userID || claims.Email != email || claims.Role != role {
			t.Errorf("claim mismatch: got %+v", claims)
		}
	})

	t.Run("ExpiredAccessToken", func(t *testing.T) {
		shortTM, _ := auth.NewTokenManager(testSecret, 50*time.Millisecond, 1*time.Hour, nil)
		tokenStr, _, _, err := shortTM.GenerateAccessToken(userID, email, role, clinicID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		_, err = shortTM.VerifyAccessToken(context.Background(), tokenStr)
		if err == nil {
			t.Fatalf("expected error for expired token, got nil")
		}
	})

	t.Run("MalformedToken", func(t *testing.T) {
		_, err := tm.VerifyAccessToken(context.Background(), "not.a.valid.jwt.token")
		if err == nil {
			t.Fatalf("expected error for malformed token, got nil")
		}
	})

	t.Run("TokenBlacklistRevocation", func(t *testing.T) {
		tokenStr, jti, _, err := tm.GenerateAccessToken(userID, email, role, clinicID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// Blacklist the JTI
		err = tm.BlacklistToken(context.Background(), jti, 1*time.Minute)
		if err != nil {
			t.Fatalf("failed to blacklist token: %v", err)
		}

		// Verify that token is now rejected
		_, err = tm.VerifyAccessToken(context.Background(), tokenStr)
		if err == nil {
			t.Fatalf("expected error for blacklisted token, got nil")
		}
	})

	t.Run("AlgorithmConfusionRejection", func(t *testing.T) {
		// Craft an unsigned "none" algorithm token
		claims := auth.Claims{
			UserID: userID,
			Email:  email,
			Role:   role,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    auth.TokenIssuer,
				Audience:  jwt.ClaimStrings{auth.TokenAudience},
				Subject:   userID,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ID:        uuid.New().String(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		unsignedStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("failed to sign token with none: %v", err)
		}

		_, err = tm.VerifyAccessToken(context.Background(), unsignedStr)
		if err == nil {
			t.Fatalf("expected algorithm confusion rejection, got nil")
		}
	})

	t.Run("RefreshTokenGenerationAndHashing", func(t *testing.T) {
		rawToken, tokenHash, err := tm.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("failed to generate refresh token: %v", err)
		}
		if len(rawToken) != 64 { // 32 bytes hex encoded = 64 chars
			t.Errorf("expected 64 character hex token, got %d", len(rawToken))
		}
		expectedHash := auth.HashToken(rawToken)
		if tokenHash != expectedHash {
			t.Errorf("expected hash %s, got %s", expectedHash, tokenHash)
		}
	})
}

func TestPasswordHashing(t *testing.T) {
	password := "SecurePanchakarmaPass2026!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == password {
		t.Fatalf("hash matches plain password")
	}

	if !auth.CheckPasswordHash(password, hash) {
		t.Fatalf("expected password to match hash")
	}

	if auth.CheckPasswordHash("WrongPassword123", hash) {
		t.Fatalf("expected wrong password to fail check")
	}
}

func TestTokenManager_SecretKeyValidation(t *testing.T) {
	_, err := auth.NewTokenManager("short_key", 15*time.Minute, 7*24*time.Hour, nil)
	if err == nil {
		t.Fatalf("expected error for secret key < 32 chars, got nil")
	}
}
