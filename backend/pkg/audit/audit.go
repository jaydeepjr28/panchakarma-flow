package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

const (
	ActionUserRegistered      = "USER_REGISTERED"
	ActionLoginSuccess        = "LOGIN_SUCCESS"
	ActionLoginFailed         = "LOGIN_FAILED"
	ActionLogout              = "LOGOUT"
	ActionTokenRotated        = "TOKEN_ROTATED"
	ActionTokenReplayDetected = "TOKEN_REPLAY_DETECTED"
	ActionStaffProvisioned    = "STAFF_PROVISIONED"
	ActionAccountDisabled     = "ACCOUNT_DISABLED"
	ActionUnauthorizedAccess  = "UNAUTHORIZED_RESOURCE_ACCESS"
)

// SecurityAuditEvent captures security-critical actions for compliance.
type SecurityAuditEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	ActorID     string    `json:"actor_id,omitempty"`
	TargetID    string    `json:"target_id,omitempty"`
	Role        string    `json:"role,omitempty"`
	ClinicID    string    `json:"clinic_id,omitempty"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Description string    `json:"description"`
	Success     bool      `json:"success"`
}

// LogSecurityEvent records a structured audit trail message.
func LogSecurityEvent(ctx context.Context, event SecurityAuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	level := slog.LevelInfo
	if !event.Success || event.Action == ActionTokenReplayDetected || event.Action == ActionUnauthorizedAccess {
		level = slog.LevelWarn
	}

	logger.WithContext(ctx).Log(
		ctx,
		level,
		"Security Audit Event",
		slog.String("audit_action", event.Action),
		slog.String("actor_id", event.ActorID),
		slog.String("target_id", event.TargetID),
		slog.String("role", event.Role),
		slog.String("clinic_id", event.ClinicID),
		slog.String("client_ip", event.IPAddress),
		slog.String("user_agent", event.UserAgent),
		slog.Bool("success", event.Success),
		slog.String("description", event.Description),
	)
}
