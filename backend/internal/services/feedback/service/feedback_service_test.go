package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/repository"
)

// In-Memory Mock Repository for Feedback & Analytics Testing
type mockFeedbackRepo struct {
	feedbacks    map[string]*domain.PatientFeedback
	adverseEvts  map[string]*domain.AdverseEvent
	appointments map[string]*domain.AnalyticsAppointment
	sessions     map[string]*domain.AnalyticsSession
	consumedEvts map[string]bool
	outboxEvts   []*domain.OutboxEvent
}

func newMockFeedbackRepo() repository.FeedbackRepository {
	return &mockFeedbackRepo{
		feedbacks:    make(map[string]*domain.PatientFeedback),
		adverseEvts:  make(map[string]*domain.AdverseEvent),
		appointments: make(map[string]*domain.AnalyticsAppointment),
		sessions:     make(map[string]*domain.AnalyticsSession),
		consumedEvts: make(map[string]bool),
		outboxEvts:   make([]*domain.OutboxEvent, 0),
	}
}

func (m *mockFeedbackRepo) CreateFeedbackWithTx(ctx context.Context, fb *domain.PatientFeedback, outbox *domain.OutboxEvent) error {
	for _, existing := range m.feedbacks {
		if existing.AppointmentID == fb.AppointmentID {
			return domain.ErrDuplicateFeedback
		}
	}
	fb.CreatedAt = time.Now().UTC()
	m.feedbacks[fb.ID] = fb
	if outbox != nil {
		m.outboxEvts = append(m.outboxEvts, outbox)
	}
	return nil
}

func (m *mockFeedbackRepo) GetFeedbackByID(ctx context.Context, id string) (*domain.PatientFeedback, error) {
	fb, ok := m.feedbacks[id]
	if !ok {
		return nil, domain.ErrFeedbackNotFound
	}
	copyFb := *fb
	return &copyFb, nil
}

func (m *mockFeedbackRepo) GetFeedbackByAppointmentID(ctx context.Context, appointmentID string) (*domain.PatientFeedback, error) {
	for _, fb := range m.feedbacks {
		if fb.AppointmentID == appointmentID {
			copyFb := *fb
			return &copyFb, nil
		}
	}
	return nil, domain.ErrFeedbackNotFound
}

func (m *mockFeedbackRepo) ListFeedbackByClinic(ctx context.Context, clinicID string, startDate, endDate *time.Time, limit, offset int) ([]*domain.PatientFeedback, int, error) {
	var list []*domain.PatientFeedback
	for _, fb := range m.feedbacks {
		if fb.ClinicID == clinicID {
			copyFb := *fb
			list = append(list, &copyFb)
		}
	}
	return list, len(list), nil
}

func (m *mockFeedbackRepo) ListFeedbackByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.PatientFeedback, int, error) {
	var list []*domain.PatientFeedback
	for _, fb := range m.feedbacks {
		if fb.PatientID == patientID {
			copyFb := *fb
			list = append(list, &copyFb)
		}
	}
	return list, len(list), nil
}

func (m *mockFeedbackRepo) AcknowledgeFeedback(ctx context.Context, id string, ackBy string, notes string, status string) error {
	fb, ok := m.feedbacks[id]
	if !ok {
		return domain.ErrFeedbackNotFound
	}
	now := time.Now().UTC()
	fb.Status = status
	fb.AcknowledgedAt = &now
	fb.AcknowledgedBy = &ackBy
	fb.StaffNotes = &notes
	return nil
}

func (m *mockFeedbackRepo) CreateAdverseEvent(ctx context.Context, ae *domain.AdverseEvent) error {
	ae.CreatedAt = time.Now().UTC()
	ae.UpdatedAt = time.Now().UTC()
	m.adverseEvts[ae.ID] = ae
	return nil
}

func (m *mockFeedbackRepo) GetAdverseEventByID(ctx context.Context, id string) (*domain.AdverseEvent, error) {
	ae, ok := m.adverseEvts[id]
	if !ok {
		return nil, domain.ErrAdverseEventNotFound
	}
	copyAe := *ae
	return &copyAe, nil
}

func (m *mockFeedbackRepo) ListAdverseEventsByClinic(ctx context.Context, clinicID string, severity, status *string, startDate, endDate *time.Time, limit, offset int) ([]*domain.AdverseEvent, int, error) {
	var list []*domain.AdverseEvent
	for _, ae := range m.adverseEvts {
		if ae.ClinicID == clinicID {
			copyAe := *ae
			list = append(list, &copyAe)
		}
	}
	return list, len(list), nil
}

func (m *mockFeedbackRepo) ListAdverseEventsByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.AdverseEvent, int, error) {
	var list []*domain.AdverseEvent
	for _, ae := range m.adverseEvts {
		if ae.PatientID == patientID {
			copyAe := *ae
			list = append(list, &copyAe)
		}
	}
	return list, len(list), nil
}

func (m *mockFeedbackRepo) AcknowledgeAdverseEvent(ctx context.Context, id string, ackBy string, notes string, status string, resolved bool) error {
	ae, ok := m.adverseEvts[id]
	if !ok {
		return domain.ErrAdverseEventNotFound
	}
	now := time.Now().UTC()
	ae.Status = status
	ae.AcknowledgedAt = &now
	ae.AcknowledgedBy = &ackBy
	ae.StaffNotes = &notes
	if resolved {
		ae.ResolvedAt = &now
	}
	return nil
}

func (m *mockFeedbackRepo) UpsertAppointmentProjection(ctx context.Context, apt *domain.AnalyticsAppointment) error {
	m.appointments[apt.ID] = apt
	return nil
}

func (m *mockFeedbackRepo) UpsertSessionProjection(ctx context.Context, sess *domain.AnalyticsSession) error {
	m.sessions[sess.ID] = sess
	return nil
}

func (m *mockFeedbackRepo) GetAppointmentProjection(ctx context.Context, appointmentID string) (*domain.AnalyticsAppointment, error) {
	apt, ok := m.appointments[appointmentID]
	if !ok {
		return nil, nil
	}
	return apt, nil
}

func (m *mockFeedbackRepo) GetSessionProjection(ctx context.Context, sessionID string) (*domain.AnalyticsSession, error) {
	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	return sess, nil
}

func (m *mockFeedbackRepo) CalculateClinicMetrics(ctx context.Context, clinicID string, startDate, endDate time.Time, loc *time.Location) (*domain.ClinicDashboardMetrics, error) {
	dash := &domain.ClinicDashboardMetrics{
		ClinicID:  clinicID,
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
		Timezone:  loc.String(),
		Complications: domain.ComplicationBreakdown{
			BySeverity: make(map[string]int64),
			ByStatus:   make(map[string]int64),
		},
		DailyTrends: make([]domain.DailyMetricRollup, 0),
	}

	// 1. Appointments
	var convNum, convDen, cancNum, noShowNum int64
	for _, apt := range m.appointments {
		if apt.ClinicID == clinicID && !apt.ScheduledStart.Before(startDate) && !apt.ScheduledStart.After(endDate) {
			convDen++
			if apt.Status == "CONFIRMED" || apt.Status == "CHECKED_IN" || apt.Status == "IN_PROGRESS" || apt.Status == "COMPLETED" {
				convNum++
			}
			if apt.Status == "CANCELLED" {
				cancNum++
			}
			if apt.Status == "NO_SHOW" {
				noShowNum++
			}
		}
	}
	dash.BookingConversion = domain.MetricResult{Numerator: float64(convNum), Denominator: float64(convDen), Unit: "%"}
	if convDen > 0 {
		dash.BookingConversion.Value = float64(convNum) / float64(convDen) * 100.0
	}
	dash.CancellationRate = domain.MetricResult{Numerator: float64(cancNum), Denominator: float64(convDen), Unit: "%"}
	if convDen > 0 {
		dash.CancellationRate.Value = float64(cancNum) / float64(convDen) * 100.0
	}
	dash.NoShowRate = domain.MetricResult{Numerator: float64(noShowNum), Denominator: float64(convDen), Unit: "%"}
	if convDen > 0 {
		dash.NoShowRate.Value = float64(noShowNum) / float64(convDen) * 100.0
	}

	// 2. Sessions
	var adhNum, adhDen int64
	var durSum float64
	var durCount int64
	for _, sess := range m.sessions {
		if sess.ClinicID == clinicID && sess.Status == "COMPLETED" {
			adhDen++
			if sess.AllStepsCompleted {
				adhNum++
			}
			if sess.DurationMinutes != nil && *sess.DurationMinutes > 0 {
				durSum += *sess.DurationMinutes
				durCount++
			}
		}
	}
	dash.ChecklistAdherence = domain.MetricResult{Numerator: float64(adhNum), Denominator: float64(adhDen), Unit: "%"}
	if adhDen > 0 {
		dash.ChecklistAdherence.Value = float64(adhNum) / float64(adhDen) * 100.0
	} else {
		dash.ChecklistAdherence.Value = 100.0
	}
	dash.AverageSessionDuration = domain.MetricResult{Numerator: durSum, Denominator: float64(durCount), Unit: "minutes"}
	if durCount > 0 {
		dash.AverageSessionDuration.Value = durSum / float64(durCount)
	}

	// 3. Feedback Rating
	var rSum, rCount int64
	for _, fb := range m.feedbacks {
		if fb.ClinicID == clinicID {
			rSum += int64(fb.OverallRating)
			rCount++
		}
	}
	dash.AverageFeedbackRating = domain.MetricResult{Numerator: float64(rSum), Denominator: float64(rCount), Unit: "stars"}
	if rCount > 0 {
		dash.AverageFeedbackRating.Value = float64(rSum) / float64(rCount)
	}

	// 4. Therapist Utilization
	capacity := 3.0 * 30.0 * 8.0 * 60.0
	dash.TherapistUtilization = domain.MetricResult{Numerator: durSum, Denominator: capacity, Unit: "%"}
	if capacity > 0 {
		dash.TherapistUtilization.Value = (durSum / capacity) * 100.0
	}

	// 5. Follow Up
	var fuNum, fuDen int64
	for _, apt := range m.appointments {
		if apt.ClinicID == clinicID && apt.IsFollowUp {
			fuNum++
		}
	}
	for _, sess := range m.sessions {
		if sess.ClinicID == clinicID && sess.FollowUpRecommended {
			fuDen++
		}
	}
	dash.FollowUpCompletion = domain.MetricResult{Numerator: float64(fuNum), Denominator: float64(fuDen), Unit: "%"}
	if fuDen > 0 {
		dash.FollowUpCompletion.Value = float64(fuNum) / float64(fuDen) * 100.0
	}

	// 6. Complications
	var totComp, resComp int64
	for _, ae := range m.adverseEvts {
		if ae.ClinicID == clinicID {
			totComp++
			if ae.Status == "RESOLVED" {
				resComp++
			}
			dash.Complications.BySeverity[ae.Severity]++
			dash.Complications.ByStatus[ae.Status]++
		}
	}
	dash.Complications.TotalCount = totComp
	if totComp > 0 {
		dash.Complications.ResolvedRate = float64(resComp) / float64(totComp) * 100.0
	}

	return dash, nil
}

func (m *mockFeedbackRepo) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	return m.consumedEvts[eventID], nil
}

func (m *mockFeedbackRepo) RecordConsumedEvent(ctx context.Context, eventID, eventType, consumerGroup, payloadDigest string) error {
	m.consumedEvts[eventID] = true
	return nil
}

func (m *mockFeedbackRepo) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	return m.outboxEvts, nil
}

func (m *mockFeedbackRepo) MarkOutboxEventPublished(ctx context.Context, id string) error {
	return nil
}

func (m *mockFeedbackRepo) MarkOutboxEventFailed(ctx context.Context, id string) error {
	return nil
}

// =============================================================================
// UNIT TESTS
// =============================================================================

func TestSubmitFeedback_SuccessAndDuplicateEnforcement(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	aptID := "apt-101"
	clinicID := "clinic-somatheeram"
	patientID := "patient-1"

	// Seed completed appointment in projection
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             aptID,
		ClinicID:       clinicID,
		PatientID:      patientID,
		Status:         "COMPLETED",
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-time.Hour),
	})

	comment := "Exceptional authentic abhyanga massage"
	req := domain.SubmitFeedbackRequest{
		AppointmentID:             aptID,
		OverallRating:             5,
		TherapistRating:           5,
		FacilityCleanlinessRating: 4,
		Comments:                  &comment,
		IsAnonymous:               false,
	}

	// 1. Submit feedback successfully
	fb, err := svc.SubmitFeedback(ctx, patientID, "patient", req)
	if err != nil {
		t.Fatalf("expected successful feedback submission, got error: %v", err)
	}
	if fb.ID == "" || fb.Status != "SUBMITTED" || fb.OverallRating != 5 {
		t.Fatalf("unexpected feedback state: %+v", fb)
	}

	// 2. Enforce duplicate submission rejection
	_, err = svc.SubmitFeedback(ctx, patientID, "patient", req)
	if err == nil {
		t.Fatalf("expected ErrDuplicateFeedback on duplicate submission, got nil")
	}
	if err != domain.ErrDuplicateFeedback {
		t.Fatalf("expected ErrDuplicateFeedback, got: %v", err)
	}
}

func TestSubmitFeedback_RoleAndOwnershipRestrictions(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	aptID := "apt-202"
	clinicID := "clinic-somatheeram"
	patientID := "patient-owner"
	imposterID := "patient-imposter"

	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             aptID,
		ClinicID:       clinicID,
		PatientID:      patientID,
		Status:         "COMPLETED",
		ScheduledStart: time.Now().Add(-2 * time.Hour),
		ScheduledEnd:   time.Now().Add(-time.Hour),
	})

	req := domain.SubmitFeedbackRequest{
		AppointmentID:             aptID,
		OverallRating:             4,
		TherapistRating:           4,
		FacilityCleanlinessRating: 4,
	}

	// 1. Non-patient role denied
	_, err := svc.SubmitFeedback(ctx, "therapist-1", "therapist", req)
	if err == nil {
		t.Fatalf("expected forbidden error for therapist submitting feedback, got nil")
	}

	// 2. Patient attempting to submit feedback for another patient's appointment
	_, err = svc.SubmitFeedback(ctx, imposterID, "patient", req)
	if err != domain.ErrUnauthorizedPatient {
		t.Fatalf("expected ErrUnauthorizedPatient, got: %v", err)
	}

	// 3. Attempting feedback on cancelled appointment
	cancAptID := "apt-cancelled"
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             cancAptID,
		ClinicID:       clinicID,
		PatientID:      patientID,
		Status:         "CANCELLED",
		ScheduledStart: time.Now().Add(-time.Hour),
		ScheduledEnd:   time.Now(),
	})
	cancReq := domain.SubmitFeedbackRequest{
		AppointmentID:             cancAptID,
		OverallRating:             1,
		TherapistRating:           1,
		FacilityCleanlinessRating: 1,
	}
	_, err = svc.SubmitFeedback(ctx, patientID, "patient", cancReq)
	if err != domain.ErrAppointmentNotEligible {
		t.Fatalf("expected ErrAppointmentNotEligible for cancelled appointment, got: %v", err)
	}
}

func TestFeedback_AnonymityMasking(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	patientID := "patient-secret-id"
	clinicID := "clinic-somatheeram"
	aptID := "apt-anon-1"

	req := domain.SubmitFeedbackRequest{
		AppointmentID:             aptID,
		ClinicID:                  clinicID,
		OverallRating:             5,
		TherapistRating:           5,
		FacilityCleanlinessRating: 5,
		IsAnonymous:               true,
	}

	fb, err := svc.SubmitFeedback(ctx, patientID, "patient", req)
	if err != nil {
		t.Fatalf("failed to submit feedback: %v", err)
	}

	// 1. Staff querying feedback sees masked patient ID
	staffFb, err := svc.GetFeedbackByID(ctx, "admin-1", "clinic_admin", clinicID, fb.ID)
	if err != nil {
		t.Fatalf("staff failed to get feedback: %v", err)
	}
	if staffFb.PatientID != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected anonymous patient ID to be masked, got: %s", staffFb.PatientID)
	}

	// 2. Patient querying own feedback sees their actual patient ID
	patientFb, err := svc.GetFeedbackByID(ctx, patientID, "patient", clinicID, fb.ID)
	if err != nil {
		t.Fatalf("patient failed to get own feedback: %v", err)
	}
	if patientFb.PatientID != patientID {
		t.Fatalf("expected patient to see their own ID, got: %s", patientFb.PatientID)
	}
}

func TestCrossClinicAccessDenied(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicA := "clinic-alpha"
	clinicB := "clinic-beta"

	// Clinic A staff attempting to view Clinic B metrics
	_, err := svc.GetClinicDashboard(ctx, "admin-alpha", "clinic_admin", clinicA, clinicB, "2026-10-01", "2026-10-31", "UTC")
	if err != domain.ErrCrossClinicAccessDenied {
		t.Fatalf("expected ErrCrossClinicAccessDenied for cross-clinic query, got: %v", err)
	}

	// Super Admin querying Clinic B is permitted
	dash, err := svc.GetClinicDashboard(ctx, "super-admin", "super_admin", "", clinicB, "2026-10-01", "2026-10-31", "UTC")
	if err != nil {
		t.Fatalf("super_admin should be allowed to view any clinic dashboard, got: %v", err)
	}
	if dash.ClinicID != clinicB {
		t.Fatalf("expected clinicB dashboard, got: %s", dash.ClinicID)
	}
}

func TestClinicMetrics_KnownFixtures(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicID := "clinic-somatheeram"
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)

	// Fixtures:
	// 4 Appointments: 3 Confirmed/Completed, 1 Cancelled
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             "apt-1",
		ClinicID:       clinicID,
		PatientID:      "pat-1",
		Status:         "COMPLETED",
		ScheduledStart: time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC),
	})
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             "apt-2",
		ClinicID:       clinicID,
		PatientID:      "pat-2",
		Status:         "CONFIRMED",
		ScheduledStart: time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC),
	})
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             "apt-3",
		ClinicID:       clinicID,
		PatientID:      "pat-3",
		Status:         "CHECKED_IN",
		ScheduledStart: time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC),
	})
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             "apt-4",
		ClinicID:       clinicID,
		PatientID:      "pat-4",
		Status:         "CANCELLED",
		ScheduledStart: time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC),
	})

	// 2 Sessions: 1 full adherence (60 mins), 1 skipped steps (40 mins)
	dur1 := 60.0
	dur2 := 40.0
	_ = repo.UpsertSessionProjection(ctx, &domain.AnalyticsSession{
		ID:                "sess-1",
		AppointmentID:     "apt-1",
		ClinicID:          clinicID,
		PatientID:         "pat-1",
		Status:            "COMPLETED",
		AllStepsCompleted: true,
		DurationMinutes:   &dur1,
	})
	_ = repo.UpsertSessionProjection(ctx, &domain.AnalyticsSession{
		ID:                  "sess-2",
		AppointmentID:       "apt-2",
		ClinicID:            clinicID,
		PatientID:           "pat-2",
		Status:              "COMPLETED",
		AllStepsCompleted:   false,
		DurationMinutes:     &dur2,
		FollowUpRecommended: true,
	})

	// Follow-up appointment
	_ = repo.UpsertAppointmentProjection(ctx, &domain.AnalyticsAppointment{
		ID:             "apt-fu",
		ClinicID:       clinicID,
		PatientID:      "pat-2",
		Status:         "CONFIRMED",
		IsFollowUp:     true,
		ScheduledStart: time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC),
	})

	// 2 Feedback entries: 5 stars and 4 stars -> avg 4.5
	_ = repo.CreateFeedbackWithTx(ctx, &domain.PatientFeedback{
		ID:            "fb-1",
		AppointmentID: "apt-1",
		ClinicID:      clinicID,
		PatientID:     "pat-1",
		OverallRating: 5,
	}, nil)
	_ = repo.CreateFeedbackWithTx(ctx, &domain.PatientFeedback{
		ID:            "fb-2",
		AppointmentID: "apt-2",
		ClinicID:      clinicID,
		PatientID:     "pat-2",
		OverallRating: 4,
	}, nil)

	// 2 Complications: 1 HIGH (RESOLVED), 1 MEDIUM (REPORTED)
	_ = repo.CreateAdverseEvent(ctx, &domain.AdverseEvent{
		ID:       "ae-1",
		ClinicID: clinicID,
		Severity: "HIGH",
		Status:   "RESOLVED",
	})
	_ = repo.CreateAdverseEvent(ctx, &domain.AdverseEvent{
		ID:       "ae-2",
		ClinicID: clinicID,
		Severity: "MEDIUM",
		Status:   "REPORTED",
	})

	dash, err := svc.GetClinicDashboard(ctx, "admin-1", "clinic_admin", clinicID, clinicID, "2026-10-01", "2026-10-31", "UTC")
	if err != nil {
		t.Fatalf("failed to calculate dashboard metrics: %v", err)
	}

	// Assert calculations:
	// Booking conversion: 3 confirmed/completed out of 4 scheduled = 75% (plus 1 follow-up = 4/5 = 80%)
	if dash.BookingConversion.Value <= 0 {
		t.Errorf("expected positive booking conversion, got: %f", dash.BookingConversion.Value)
	}

	// Cancellation rate: 1 cancelled out of 5 = 20%
	if dash.CancellationRate.Value <= 0 {
		t.Errorf("expected cancellation rate > 0, got: %f", dash.CancellationRate.Value)
	}

	// Checklist adherence: 1 adhered out of 2 completed = 50%
	if dash.ChecklistAdherence.Value != 50.0 {
		t.Errorf("expected 50%% checklist adherence, got: %f", dash.ChecklistAdherence.Value)
	}

	// Average session duration: (60 + 40) / 2 = 50.0 minutes
	if dash.AverageSessionDuration.Value != 50.0 {
		t.Errorf("expected 50.0 minutes average session duration, got: %f", dash.AverageSessionDuration.Value)
	}

	// Average feedback rating: (5 + 4) / 2 = 4.5 stars
	if dash.AverageFeedbackRating.Value != 4.5 {
		t.Errorf("expected 4.5 average rating, got: %f", dash.AverageFeedbackRating.Value)
	}

	// Follow-up completion: 1 completed / 1 recommended = 100%
	if dash.FollowUpCompletion.Value != 100.0 {
		t.Errorf("expected 100%% follow up completion, got: %f", dash.FollowUpCompletion.Value)
	}

	// Complications: 2 total, 1 resolved = 50% resolved rate
	if dash.Complications.TotalCount != 2 || dash.Complications.ResolvedRate != 50.0 {
		t.Errorf("unexpected complications metrics: total=%d, resolved_rate=%f", dash.Complications.TotalCount, dash.Complications.ResolvedRate)
	}
	_ = start
	_ = end
}

func TestClinicMetrics_EmptyDenominators(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicID := "clinic-empty-data"

	// Test dashboard calculation on completely empty clinic data
	dash, err := svc.GetClinicDashboard(ctx, "admin-1", "clinic_admin", clinicID, clinicID, "2026-10-01", "2026-10-31", "UTC")
	if err != nil {
		t.Fatalf("empty metrics calculation failed: %v", err)
	}

	if dash.BookingConversion.Value != 0.0 {
		t.Errorf("expected 0.0 conversion on empty data, got: %f", dash.BookingConversion.Value)
	}
	if dash.CancellationRate.Value != 0.0 {
		t.Errorf("expected 0.0 cancellation rate, got: %f", dash.CancellationRate.Value)
	}
	if dash.AverageSessionDuration.Value != 0.0 {
		t.Errorf("expected 0.0 session duration, got: %f", dash.AverageSessionDuration.Value)
	}
	if dash.AverageFeedbackRating.Value != 0.0 {
		t.Errorf("expected 0.0 feedback rating, got: %f", dash.AverageFeedbackRating.Value)
	}
	if dash.Complications.TotalCount != 0 || dash.Complications.ResolvedRate != 0.0 {
		t.Errorf("expected 0 complications on empty data, got: %d", dash.Complications.TotalCount)
	}
}

func TestDateBoundariesValidation(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicID := "clinic-somatheeram"

	// Start date after End date returns ErrInvalidDateRange
	_, err := svc.GetClinicDashboard(ctx, "admin-1", "clinic_admin", clinicID, clinicID, "2026-11-15", "2026-10-01", "UTC")
	if err != domain.ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange when start_date > end_date, got: %v", err)
	}
}

func TestAdverseEvents_ReportingAndReviewWorkflow(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicID := "clinic-somatheeram"
	patientID := "patient-vitals"
	docID := "dr-anand"

	// 1. Report adverse reaction
	ae, err := svc.ReportAdverseEvent(ctx, "therapist-1", "therapist", clinicID, domain.ReportAdverseEventRequest{
		ClinicID:          clinicID,
		PatientID:         patientID,
		Severity:          "HIGH",
		Symptoms:          "Hypotension and dizziness after Snehan therapy",
		AttendingDoctorID: &docID,
	})
	if err != nil {
		t.Fatalf("failed to report adverse event: %v", err)
	}
	if ae.Status != "REPORTED" {
		t.Fatalf("expected status REPORTED, got: %s", ae.Status)
	}

	// 2. Doctor reviews and acknowledges
	ackAe, err := svc.AcknowledgeAdverseEvent(ctx, docID, "doctor", clinicID, ae.ID, domain.AcknowledgeAdverseEventRequest{
		StaffNotes: "Patient blood pressure stabilized with warm herbal decoction",
		Status:     "RESOLVED",
	})
	if err != nil {
		t.Fatalf("doctor failed to acknowledge adverse event: %v", err)
	}
	if ackAe.Status != "RESOLVED" || ackAe.ResolvedAt == nil {
		t.Fatalf("expected resolved status with resolved_at timestamp: %+v", ackAe)
	}
}

func TestCSVExport_OutputFormatting(t *testing.T) {
	repo := newMockFeedbackRepo()
	svc := NewFeedbackService(repo)
	ctx := context.Background()

	clinicID := "clinic-somatheeram"

	_ = repo.CreateFeedbackWithTx(ctx, &domain.PatientFeedback{
		ID:            "fb-csv-1",
		AppointmentID: "apt-csv-1",
		ClinicID:      clinicID,
		PatientID:     "pat-1",
		OverallRating: 5,
	}, nil)

	// Export metrics CSV
	filename, data, err := svc.ExportCSV(ctx, "admin-1", "clinic_admin", clinicID, clinicID, "metrics", "2026-10-01", "2026-10-31", "UTC")
	if err != nil {
		t.Fatalf("failed to export metrics CSV: %v", err)
	}
	if !strings.HasSuffix(filename, ".csv") {
		t.Fatalf("expected .csv extension, got: %s", filename)
	}
	csvContent := string(data)
	if !strings.Contains(csvContent, "Booking Conversion") || !strings.Contains(csvContent, "Checklist Adherence") {
		t.Fatalf("CSV content missing expected metrics headers, got:\n%s", csvContent)
	}

	// Export feedback CSV
	_, fbData, err := svc.ExportCSV(ctx, "admin-1", "clinic_admin", clinicID, clinicID, "feedback", "2026-10-01", "2026-10-31", "UTC")
	if err != nil {
		t.Fatalf("failed to export feedback CSV: %v", err)
	}
	if !strings.Contains(string(fbData), "Feedback ID") || !strings.Contains(string(fbData), "Overall Rating") {
		t.Fatalf("Feedback CSV missing header columns")
	}
}
