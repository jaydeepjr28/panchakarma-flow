package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

func TestPostgresClinicRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	testDB := database.NewTestDB(db)
	repo := repository.NewPostgresClinicRepository(testDB)
	ctx := context.Background()

	t.Run("CreateClinic_Success", func(t *testing.T) {
		clinic := &domain.Clinic{
			ID:                 uuid.New().String(),
			Name:               "Kerala Ayurvedic Retreat",
			RegistrationNumber: "AYUR-KL-2026-001",
			AddressStreet:      "45 Temple Road",
			AddressCity:        "Kochi",
			AddressState:       "Kerala",
			AddressPostalCode:  "682001",
			ContactPhone:       "+914842345678",
			ContactEmail:       "info@keralaayur.com",
			Description:        "Premier Panchakarma Healing Sanctuary",
			SupportedLanguages: []string{"English", "Malayalam", "Hindi"},
			OperatingHours:     json.RawMessage(`{"monday":[{"open":"08:00","close":"18:00"}]}`),
			Timezone:           "Asia/Kolkata",
			IsActive:           true,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO clinics").
			WithArgs(
				clinic.ID, clinic.Name, clinic.RegistrationNumber, clinic.AddressStreet, clinic.AddressCity,
				clinic.AddressState, clinic.AddressPostalCode, clinic.ContactPhone, clinic.ContactEmail,
				clinic.Description, sqlmock.AnyArg(), clinic.OperatingHours, clinic.Timezone,
				clinic.IsActive, clinic.CreatedAt, clinic.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateClinic(ctx, clinic)
		if err != nil {
			t.Fatalf("unexpected error creating clinic: %v", err)
		}
	})

	t.Run("GetClinicByID_Found", func(t *testing.T) {
		id := uuid.New().String()
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{
			"id", "name", "registration_number", "address_street", "address_city", "address_state", "address_postal_code",
			"contact_phone", "contact_email", "description", "supported_languages", "operating_hours", "timezone", "is_active", "created_at", "updated_at",
		}).AddRow(
			id, "Rishikesh Ayur Ashram", "AYUR-UT-2026-002", "Ganga Kinare", "Rishikesh", "Uttarakhand", "249201",
			"+911352456789", "ashram@rishikesh.org", "Ganges riverside holistic rejuvenation", "{English,Hindi}",
			[]byte(`{"monday":[{"open":"07:00","close":"19:00"}]}`), "Asia/Kolkata", true, now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM clinics WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		clinic, err := repo.GetClinicByID(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clinic == nil || clinic.ID != id || clinic.Timezone != "Asia/Kolkata" {
			t.Fatalf("clinic mismatch: got %+v", clinic)
		}
	})

	t.Run("TreatmentRoomAndEquipment", func(t *testing.T) {
		clinicID := uuid.New().String()
		room := &domain.TreatmentRoom{
			ID:                      uuid.New().String(),
			ClinicID:                clinicID,
			Name:                    "Droni Suite 1",
			RoomType:                "panchakarma_specialized",
			Capacity:                1,
			SanitationBufferMinutes: 20,
			IsActive:                true,
			CreatedAt:               time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO treatment_rooms").
			WithArgs(room.ID, room.ClinicID, room.Name, room.RoomType, room.Capacity, room.SanitationBufferMinutes, room.IsActive, room.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateRoom(ctx, room)
		if err != nil {
			t.Fatalf("failed to create room: %v", err)
		}

		equip := &domain.RoomEquipment{
			ID:            uuid.New().String(),
			RoomID:        room.ID,
			EquipmentType: "droni_table",
			SerialNumber:  "DRONI-TEAK-2026-01",
			IsOperational: true,
		}

		mock.ExpectExec("INSERT INTO room_equipment").
			WithArgs(equip.ID, equip.RoomID, equip.EquipmentType, equip.SerialNumber, equip.IsOperational).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.AddRoomEquipment(ctx, equip)
		if err != nil {
			t.Fatalf("failed to add equipment: %v", err)
		}
	})

	t.Run("DoctorAndTherapistProfiles", func(t *testing.T) {
		clinicID := uuid.New().String()
		now := time.Now().UTC()

		doc := &domain.Doctor{
			ID:                uuid.New().String(),
			UserID:            uuid.New().String(),
			ClinicID:          clinicID,
			Qualifications:    "BAMS, MD (Panchakarma)",
			Specialties:       []string{"Virechana", "Vamana", "Nasya"},
			ExperienceYears:   12,
			MaxSessionsPerDay: 8,
			IsActive:          true,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		mock.ExpectExec("INSERT INTO doctors").
			WithArgs(doc.ID, doc.UserID, doc.ClinicID, doc.Qualifications, sqlmock.AnyArg(), doc.ExperienceYears, doc.MaxSessionsPerDay, doc.IsActive, doc.CreatedAt, doc.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateDoctor(ctx, doc)
		if err != nil {
			t.Fatalf("failed to create doctor: %v", err)
		}

		therapist := &domain.Therapist{
			ID:              uuid.New().String(),
			UserID:          uuid.New().String(),
			ClinicID:        clinicID,
			Specializations: []string{"Abhyanga", "Shirodhara", "Kizhi"},
			ExperienceYears: 6,
			Gender:          "male",
			IsActive:        true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		mock.ExpectExec("INSERT INTO therapists").
			WithArgs(therapist.ID, therapist.UserID, therapist.ClinicID, sqlmock.AnyArg(), therapist.ExperienceYears, therapist.Gender, therapist.IsActive, therapist.CreatedAt, therapist.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.CreateTherapist(ctx, therapist)
		if err != nil {
			t.Fatalf("failed to create therapist: %v", err)
		}
	})

	t.Run("DoctorScheduleAndLeaves", func(t *testing.T) {
		docID := uuid.New().String()

		sch := &domain.DoctorSchedule{
			ID:                  uuid.New().String(),
			DoctorID:            docID,
			DayOfWeek:           1, // Monday
			StartTime:           "09:00",
			EndTime:             "17:00",
			SlotDurationMinutes: 60,
			IsActive:            true,
		}

		mock.ExpectExec("INSERT INTO doctor_schedules").
			WithArgs(sch.ID, sch.DoctorID, sch.DayOfWeek, sch.StartTime, sch.EndTime, sch.SlotDurationMinutes, sch.IsActive).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateSchedule(ctx, sch)
		if err != nil {
			t.Fatalf("failed to create schedule: %v", err)
		}

		now := time.Now().UTC()
		leave := &domain.DoctorLeave{
			ID:        uuid.New().String(),
			DoctorID:  docID,
			StartTime: now.Add(24 * time.Hour),
			EndTime:   now.Add(48 * time.Hour),
			Reason:    "National Ayurveda Conference",
			Status:    "APPROVED",
			CreatedAt: now,
		}

		mock.ExpectExec("INSERT INTO doctor_leaves").
			WithArgs(leave.ID, leave.DoctorID, leave.StartTime, leave.EndTime, leave.Reason, leave.Status, leave.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.CreateLeave(ctx, leave)
		if err != nil {
			t.Fatalf("failed to create leave: %v", err)
		}
	})

	t.Run("TherapyPackagesAndSteps", func(t *testing.T) {
		clinicID := uuid.New().String()
		now := time.Now().UTC()

		pkg := &domain.TherapyPackage{
			ID:                       uuid.New().String(),
			ClinicID:                 clinicID,
			Name:                     "Shodhan Shirodhara 7-Day",
			Category:                 "stress_relief",
			Description:              "Traditional buttermilk and medicated oil dripping for insomnia and Vata pacification",
			PriceCents:               3500000,
			TotalSessions:            7,
			DurationMinutes:          90,
			RecommendedIntervalsDays: 1,
			Version:                  1,
			IsPublished:              false,
			IsActive:                 true,
			CreatedAt:                now,
			UpdatedAt:                now,
		}

		mock.ExpectExec("INSERT INTO therapy_packages").
			WithArgs(
				pkg.ID, pkg.ClinicID, pkg.Name, pkg.Category, pkg.Description, pkg.PriceCents, pkg.TotalSessions,
				pkg.DurationMinutes, pkg.RecommendedIntervalsDays, pkg.Version, pkg.IsPublished, pkg.IsActive, pkg.CreatedAt, pkg.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreatePackage(ctx, pkg)
		if err != nil {
			t.Fatalf("failed to create package: %v", err)
		}

		steps := []domain.PackageStep{
			{
				ID:                uuid.New().String(),
				PackageID:         pkg.ID,
				StepOrder:         1,
				Name:              "Poorvakarma - Siro Abhyanga",
				Description:       "Gentle head massage with Brahmi oil",
				DurationMinutes:   15,
				Instructions:      []string{"Inspect scalp", "Warm Brahmi oil to 38C"},
				RequiredEquipment: []string{"brass_katori"},
			},
			{
				ID:                uuid.New().String(),
				PackageID:         pkg.ID,
				StepOrder:         2,
				Name:              "Pradhanakarma - Taila Dhara",
				Description:       "Continuous rhythmic oscillation of medicated oil on Ajna chakra",
				DurationMinutes:   45,
				Instructions:      []string{"Maintain 4-inch stream height", "Monitor patient pulse"},
				RequiredEquipment: []string{"shirodhara_stand", "droni_table"},
			},
		}

		mock.ExpectExec("DELETE FROM therapy_package_steps WHERE package_id = \\$1").
			WithArgs(pkg.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		for _, s := range steps {
			mock.ExpectExec("INSERT INTO therapy_package_steps").
				WithArgs(s.ID, pkg.ID, s.StepOrder, s.Name, s.Description, s.DurationMinutes, sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}

		err = repo.SavePackageSteps(ctx, pkg.ID, steps)
		if err != nil {
			t.Fatalf("failed to save package steps: %v", err)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
