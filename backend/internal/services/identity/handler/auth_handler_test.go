package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	identityv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/identity/v1"
)

// In-memory repository for HTTP acceptance testing
type testUserRepo struct {
	mu       sync.RWMutex
	users    map[string]*domain.User
	profiles map[string]*domain.PatientProfile
	sessions map[string]*domain.RefreshSession
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{
		users:    make(map[string]*domain.User),
		profiles: make(map[string]*domain.PatientProfile),
		sessions: make(map[string]*domain.RefreshSession),
	}
}

func (m *testUserRepo) CreateUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	return nil
}

func (m *testUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, nil
}

func (m *testUserRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[id], nil
}

func (m *testUserRepo) UpdateUserStatus(ctx context.Context, id string, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, exists := m.users[id]; exists {
		u.IsActive = isActive
	}
	return nil
}

func (m *testUserRepo) CreatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *testUserRepo) GetPatientProfile(ctx context.Context, userID string) (*domain.PatientProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiles[userID], nil
}

func (m *testUserRepo) UpdatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *testUserRepo) CreateRefreshSession(ctx context.Context, session *domain.RefreshSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.TokenHash] = session
	return nil
}

func (m *testUserRepo) GetRefreshSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[tokenHash], nil
}

func (m *testUserRepo) RevokeRefreshSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ID == sessionID {
			s.IsRevoked = true
		}
	}
	return nil
}

func (m *testUserRepo) RevokeRefreshFamily(ctx context.Context, familyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.FamilyID == familyID {
			s.IsRevoked = true
		}
	}
	return nil
}

func setupTestRouter(t *testing.T) (*gin.Engine, *auth.TokenManager, service.AuthService, *testUserRepo) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	repo := newTestUserRepo()
	tm, err := auth.NewTokenManager("panchakarma_super_secret_test_key_at_least_32_bytes", 15*time.Minute, 7*24*time.Hour, nil)
	if err != nil {
		t.Fatalf("failed to init TokenManager: %v", err)
	}

	svc := service.NewAuthService(repo, tm)
	h := handler.NewAuthHandler(svc, tm)

	apiV1 := router.Group("/api/v1")
	h.RegisterRoutes(apiV1)

	return router, tm, svc, repo
}

func TestHTTP_PatientRegistration(t *testing.T) {
	router, _, _, _ := setupTestRouter(t)

	t.Run("ValidRegistration_201Created", func(t *testing.T) {
		body := map[string]any{
			"email":                   "patient.new@example.com",
			"phone":                   "+919876543210",
			"password":                "SecurePass123!",
			"full_name":               "Aarav Sharma",
			"date_of_birth":           "1988-08-20",
			"biological_sex":          "male",
			"blood_group":             "A+",
			"emergency_contact_name":  "Pooja Sharma",
			"emergency_contact_phone": "+919876543211",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var res struct {
			Success bool `json:"success"`
			Data    struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				User         struct {
					Role string `json:"role"`
				} `json:"user"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)

		if !res.Success || res.Data.AccessToken == "" || res.Data.RefreshToken == "" {
			t.Fatalf("invalid response structure: %+v", res)
		}
		if res.Data.User.Role != "patient" {
			t.Errorf("expected role 'patient', got %s", res.Data.User.Role)
		}
	})

	t.Run("PrivilegeEscalationAttempt_403Forbidden", func(t *testing.T) {
		body := map[string]any{
			"email":                   "hacker@example.com",
			"phone":                   "+919876543210",
			"password":                "SecurePass123!",
			"role":                    "clinic_admin", // Attempting privilege escalation
			"full_name":               "Attacker",
			"date_of_birth":           "1990-01-01",
			"biological_sex":          "male",
			"emergency_contact_name":  "Nobody",
			"emergency_contact_phone": "+919876543211",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for privilege escalation, got %d", w.Code)
		}
	})

	t.Run("DuplicateRegistration_409Conflict", func(t *testing.T) {
		body := map[string]any{
			"email":                   "duplicate.http@example.com",
			"phone":                   "+919876543210",
			"password":                "SecurePass123!",
			"full_name":               "Person",
			"date_of_birth":           "1990-01-01",
			"biological_sex":          "female",
			"emergency_contact_name":  "Kin",
			"emergency_contact_phone": "+919876543211",
		}
		jsonBytes, _ := json.Marshal(body)

		// First request: OK
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBytes))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		if w1.Code != http.StatusCreated {
			t.Fatalf("first registration failed: %d", w1.Code)
		}

		// Second request: Conflict
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBytes))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d", w2.Code)
		}
	})
}

func TestHTTP_LoginAndRefreshCycle(t *testing.T) {
	router, _, _, repo := setupTestRouter(t)

	// Pre-register user
	regBody := map[string]any{
		"email":                   "lifecycle@example.com",
		"phone":                   "+919876543210",
		"password":                "ValidPassword123!",
		"full_name":               "Lifecycle User",
		"date_of_birth":           "1995-10-10",
		"biological_sex":          "female",
		"emergency_contact_name":  "Mother",
		"emergency_contact_phone": "+919876543211",
	}
	regBytes, _ := json.Marshal(regBody)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, regReq)

	var regRes struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wReg.Body.Bytes(), &regRes)
	userID := regRes.Data.User.ID

	t.Run("ValidLogin_200OK", func(t *testing.T) {
		loginBody := map[string]string{
			"email":    "lifecycle@example.com",
			"password": "ValidPassword123!",
		}
		loginBytes, _ := json.Marshal(loginBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("InvalidPassword_401Unauthorized", func(t *testing.T) {
		loginBody := map[string]string{
			"email":    "lifecycle@example.com",
			"password": "WrongPassword!",
		}
		loginBytes, _ := json.Marshal(loginBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DisabledAccount_403Forbidden", func(t *testing.T) {
		_ = repo.UpdateUserStatus(context.Background(), userID, false)

		loginBody := map[string]string{
			"email":    "lifecycle@example.com",
			"password": "ValidPassword123!",
		}
		loginBytes, _ := json.Marshal(loginBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for disabled account, got %d", w.Code)
		}

		_ = repo.UpdateUserStatus(context.Background(), userID, true)
	})

	t.Run("TokenRotationAndReplayDetection", func(t *testing.T) {
		// 1. Initial valid refresh token from registration
		refreshBody := map[string]string{
			"refresh_token": regRes.Data.RefreshToken,
		}
		refreshBytes, _ := json.Marshal(refreshBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refreshBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on token rotation, got %d: %s", w.Code, w.Body.String())
		}

		var rotRes struct {
			Data struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &rotRes)

		// 2. Replay attack: Submit old refresh token again
		replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(refreshBytes))
		replayReq.Header.Set("Content-Type", "application/json")
		wReplay := httptest.NewRecorder()
		router.ServeHTTP(wReplay, replayReq)

		if wReplay.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized on replay attack, got %d", wReplay.Code)
		}

		// 3. New token in same family must now be revoked due to replay detection
		subsequentBody, _ := json.Marshal(map[string]string{"refresh_token": rotRes.Data.RefreshToken})
		subsequentReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(subsequentBody))
		subsequentReq.Header.Set("Content-Type", "application/json")
		wSubsequent := httptest.NewRecorder()
		router.ServeHTTP(wSubsequent, subsequentReq)

		if wSubsequent.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for invalidated family, got %d", wSubsequent.Code)
		}
	})
}

func TestHTTP_LogoutAndBlacklisting(t *testing.T) {
	router, _, _, _ := setupTestRouter(t)

	// Register user to obtain access token
	regBody := map[string]any{
		"email":                   "logout.http@example.com",
		"phone":                   "+919876543210",
		"password":                "Pass123456!",
		"full_name":               "Logout Tester",
		"date_of_birth":           "1990-01-01",
		"biological_sex":          "male",
		"emergency_contact_name":  "Kin",
		"emergency_contact_phone": "+919876543211",
	}
	regBytes, _ := json.Marshal(regBody)
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(regBytes))
	regReq.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, regReq)

	var regRes struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wReg.Body.Bytes(), &regRes)
	accessToken := regRes.Data.AccessToken

	// Verify authenticated route works before logout
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)
	wMe := httptest.NewRecorder()
	router.ServeHTTP(wMe, meReq)
	if wMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK before logout, got %d", wMe.Code)
	}

	// Perform logout
	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": regRes.Data.RefreshToken})
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBuffer(logoutBody))
	logoutReq.Header.Set("Authorization", "Bearer "+accessToken)
	logoutReq.Header.Set("Content-Type", "application/json")
	wLogout := httptest.NewRecorder()
	router.ServeHTTP(wLogout, logoutReq)

	if wLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on logout, got %d", wLogout.Code)
	}

	// Verify access token is now blacklisted on subsequent request
	meReqAfter := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReqAfter.Header.Set("Authorization", "Bearer "+accessToken)
	wMeAfter := httptest.NewRecorder()
	router.ServeHTTP(wMeAfter, meReqAfter)

	if wMeAfter.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for blacklisted token, got %d", wMeAfter.Code)
	}
}

func TestHTTP_AuthorizationAndResourceOwnership(t *testing.T) {
	router, tm, _, _ := setupTestRouter(t)

	// Create two patients
	patientA_ID := uuid.New().String()
	patientB_ID := uuid.New().String()

	tokenA, _, _, _ := tm.GenerateAccessToken(patientA_ID, "patientA@example.com", "patient", "")
	tokenB, _, _, _ := tm.GenerateAccessToken(patientB_ID, "patientB@example.com", "patient", "")
	doctorToken, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "doctor@clinic.com", "doctor", "clinic-1")

	t.Run("PatientA_CannotAccess_PatientB_Profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/patients/"+patientB_ID+"/profile", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA) // Patient A token
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden when Patient A accesses Patient B, got %d", w.Code)
		}
	})

	t.Run("PatientA_CanAccess_OwnProfile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/patients/"+patientA_ID+"/profile", nil)
		req.Header.Set("Authorization", "Bearer "+tokenA)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 404 is acceptable here since profile record not seeded, but NOT 403 Forbidden!
		if w.Code == http.StatusForbidden {
			t.Fatalf("expected authorization pass (got 403 Forbidden)")
		}
	})

	t.Run("Doctor_CanAccess_PatientProfile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/patients/"+patientA_ID+"/profile", nil)
		req.Header.Set("Authorization", "Bearer "+doctorToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusForbidden {
			t.Fatalf("expected doctor to have access to patient profile, got 403 Forbidden")
		}
	})

	t.Run("Patient_CannotProvisionStaff", func(t *testing.T) {
		body := map[string]any{
			"email":     "staff@example.com",
			"phone":     "+919876543210",
			"password":  "Pass123456!",
			"role":      "therapist",
			"full_name": "New Therapist",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/provision-staff", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+tokenB) // Patient token
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for staff provisioning by patient, got %d", w.Code)
		}
	})
}

func TestHTTP_RateLimiting(t *testing.T) {
	router, _, _, _ := setupTestRouter(t)

	// Endpoint /api/v1/auth/login has rate limit: 5 requests per minute
	for i := 1; i <= 6; i++ {
		loginBody, _ := json.Marshal(map[string]string{
			"email":    "ratelimit@example.com",
			"password": "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.195:12345" // Specific client IP

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i <= 5 {
			// First 5 requests should get 401 Unauthorized (wrong creds), not 429
			if w.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d was unexpectedly rate limited", i)
			}
		} else {
			// 6th request should be rate limited with 429
			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("expected 6th request to be 429 Too Many Requests, got %d", w.Code)
			}
			retryAfter := w.Header().Get("Retry-After")
			if retryAfter == "" {
				t.Errorf("expected Retry-After header on 429 response")
			}
		}
	}
}

func TestGRPC_IdentityService(t *testing.T) {
	_, tm, svc, repo := setupTestRouter(t)
	grpcHandler := handler.NewGRPCHandler(svc)
	ctx := context.Background()

	// 1. Generate valid token
	tokenStr, _, _, err := tm.GenerateAccessToken("user-123", "user@example.com", "patient", "")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	t.Run("ValidateToken_Valid", func(t *testing.T) {
		res, err := grpcHandler.ValidateToken(ctx, &identityv1.ValidateTokenRequest{
			AccessToken: tokenStr,
		})
		if err != nil {
			t.Fatalf("gRPC error: %v", err)
		}
		if !res.IsValid || res.UserId != "user-123" || res.Role != "patient" {
			t.Errorf("validation response mismatch: %+v", res)
		}
	})

	t.Run("ValidateToken_Invalid", func(t *testing.T) {
		res, err := grpcHandler.ValidateToken(ctx, &identityv1.ValidateTokenRequest{
			AccessToken: "invalid.jwt.token",
		})
		if err != nil {
			t.Fatalf("unexpected gRPC error: %v", err)
		}
		if res.IsValid {
			t.Errorf("expected invalid token to return is_valid=false")
		}
	})

	t.Run("GetPatientSummary_Success", func(t *testing.T) {
		patientID := uuid.New().String()
		_ = repo.CreatePatientProfile(ctx, &domain.PatientProfile{
			UserID:                patientID,
			FullName:              "Vaidya Patient",
			EmergencyContactPhone: "+919876543210",
			BloodGroup:            "AB+",
		})

		res, err := grpcHandler.GetPatientSummary(ctx, &identityv1.GetPatientSummaryRequest{
			PatientId: patientID,
		})
		if err != nil {
			t.Fatalf("gRPC summary failed: %v", err)
		}
		if res.FullName != "Vaidya Patient" || res.BloodGroup != "AB+" {
			t.Errorf("summary mismatch: %+v", res)
		}
	})
}
