# ==============================================================================
# PanchakarmaFlow — Phase 8 Feedback, Adverse Events & Analytics E2E Test Suite
# Tests:
#  1. Health & Gateway Routing
#  2. Patient Registration, Feedback Submission & Duplicate Prevention (409)
#  3. Role Restrictions & Anonymity Masking
#  4. Clinic Staff Review & Acknowledgment Workflow
#  5. Adverse Events / Complications Reporting & Clinical Resolution
#  6. Clinic Dashboard Metrics (Conversion, Cancellation, Adherence, Duration, Ratings, Utilization, Follow-Up, Complications)
#  7. Edge Cases: Empty Ranges (Zero Denominators) & Date Boundary Inversion
#  8. Multi-Tenancy & Cross-Clinic Access Denial (403 Forbidden)
#  9. CSV Data Exports (Metrics, Feedback, Adverse Events)
# ==============================================================================

$ErrorActionPreference = "Continue"
$BaseUrl = "http://localhost:8080/api/v1"
$DirectAnalyticsUrl = "http://localhost:8086"
$PassedTests = 0
$FailedTests = 0

function Assert-Step($name, $condition, $details = "") {
    if ($condition) {
        Write-Host "  [PASS] $name" -ForegroundColor Green
        $script:PassedTests++
    } else {
        Write-Host "  [FAIL] $name" -ForegroundColor Red
        if ($details) { Write-Host "         Details: $details" -ForegroundColor Yellow }
        $script:FailedTests++
    }
}

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  PANCHAKARMA FLOW -- PHASE 8 FEEDBACK & ANALYTICS E2E SUITE" -ForegroundColor White
Write-Host "==========================================================================" -ForegroundColor Cyan

# ------------------------------------------------------------------------------
# 1. HEALTH AND SERVICE CONNECTIVITY
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 1] Health & Connectivity ---" -ForegroundColor Yellow

try {
    $healthRes = Invoke-RestMethod -Uri "$DirectAnalyticsUrl/health/ready" -Method Get
    Assert-Step "1.1 Feedback & Analytics direct /health/ready is UP" ($healthRes.status -eq "UP" -and $healthRes.components.database.status -eq "UP")
} catch {
    Assert-Step "1.1 Feedback & Analytics direct /health/ready" $false $_.Exception.Message
}

try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/feedback" -Method Get
    Assert-Step "1.2 Gateway routes /feedback and requires authentication (401)" $false "Expected 401"
} catch {
    $statusCode = [int]$_.Exception.Response.StatusCode
    Assert-Step "1.2 Gateway routes /feedback and requires authentication (401)" ($statusCode -eq 401) "Status: $statusCode"
}

# ------------------------------------------------------------------------------
# 2. USER AUTHENTICATION & SETUP
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 2] Authentication & Role Setup ---" -ForegroundColor Yellow

# 2.1 Dynamic Patient Registration
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$testPatientEmail = "feedback.patient.$timestamp@example.com"
$testPatientPass = "PatientPass123!"
$phoneSuffix = ($timestamp.ToString().Substring($timestamp.ToString().Length - 4))

$regBody = @{
    email = $testPatientEmail
    password = $testPatientPass
    full_name = "Feedback Test Patient"
    phone = "+91987654$phoneSuffix"
    date_of_birth = "1994-08-12"
    biological_sex = "male"
    emergency_contact_name = "Emergency Kin"
    emergency_contact_phone = "+919876540001"
} | ConvertTo-Json

$patientToken = $null
$patientId = $null
try {
    $regRes = Invoke-RestMethod -Uri "$BaseUrl/auth/register" -Method Post -Body $regBody -ContentType "application/json"
    $patientToken = $regRes.data.access_token
    $patientId = $regRes.data.user.id
    Assert-Step "2.1 Test patient registered and JWT obtained" ($patientToken -ne $null -and $patientId -ne $null)
} catch {
    Assert-Step "2.1 Test patient registration" $false $_.Exception.Message
}

# 2.2 Clinic Admin Login
$adminToken = $null
$adminClinicId = $null
try {
    $adminRes = Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body (@{
        email = "admin.ayur@somatheeram.in"
        password = "AdminPass123!"
    } | ConvertTo-Json) -ContentType "application/json"
    $adminToken = $adminRes.data.access_token
    $adminClinicId = $adminRes.data.user.clinic_id
    Assert-Step "2.2 Clinic Admin authentication succeeds" ($adminToken -ne $null -and $adminClinicId -ne $null)
} catch {
    Assert-Step "2.2 Clinic Admin authentication" $false $_.Exception.Message
}

# 2.3 Doctor Login
$doctorToken = $null
$doctorId = $null
try {
    $docRes = Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body (@{
        email = "dr.anand@somatheeram.in"
        password = "DoctorPass123!"
    } | ConvertTo-Json) -ContentType "application/json"
    $doctorToken = $docRes.data.access_token
    $doctorId = $docRes.data.user.id
    Assert-Step "2.3 Doctor authentication succeeds" ($doctorToken -ne $null)
} catch {
    Assert-Step "2.3 Doctor authentication" $false $_.Exception.Message
}

# 2.4 Therapist Login
$therapistToken = $null
try {
    $thRes = Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body (@{
        email = "ramesh.vaidya@somatheeram.in"
        password = "TherapistPass123!"
    } | ConvertTo-Json) -ContentType "application/json"
    $therapistToken = $thRes.data.access_token
    Assert-Step "2.4 Therapist authentication succeeds" ($therapistToken -ne $null)
} catch {
    Assert-Step "2.4 Therapist authentication" $false $_.Exception.Message
}

$patientHeaders = @{ Authorization = "Bearer $patientToken" }
$adminHeaders = @{ Authorization = "Bearer $adminToken" }
$docHeaders = @{ Authorization = "Bearer $doctorToken" }
$therapistHeaders = @{ Authorization = "Bearer $therapistToken" }

# ------------------------------------------------------------------------------
# 3. OPERATIONAL DATA RECONCILIATION & SEEDING
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 3] Operational Data Reconciliation ---" -ForegroundColor Yellow

$testAptId = [System.Guid]::NewGuid().ToString()
$testSessId = [System.Guid]::NewGuid().ToString()
$anonAptId = [System.Guid]::NewGuid().ToString()
$nowUtc = (Get-Date).ToUniversalTime()
$startedUtc = $nowUtc.AddMinutes(-60).ToString("o")
$completedUtc = $nowUtc.ToString("o")

$reconcileBody = @{
    appointments = @(
        @{
            id = $testAptId
            clinic_id = $adminClinicId
            patient_id = $patientId
            doctor_id = $doctorId
            status = "COMPLETED"
            scheduled_start = $nowUtc.AddHours(-2).ToString("o")
            scheduled_end = $nowUtc.AddHours(-1).ToString("o")
            is_follow_up = $false
        },
        @{
            id = "a0000000-0000-0000-0000-000000000002"
            clinic_id = $adminClinicId
            patient_id = $patientId
            status = "CONFIRMED"
            scheduled_start = $nowUtc.AddDays(1).ToString("o")
            scheduled_end = $nowUtc.AddDays(1).AddHours(2).ToString("o")
            is_follow_up = $true
        },
        @{
            id = "a0000000-0000-0000-0000-000000000003"
            clinic_id = $adminClinicId
            patient_id = "p0000000-0000-0000-0000-000000000099"
            status = "CANCELLED"
            scheduled_start = $nowUtc.AddDays(-3).ToString("o")
            scheduled_end = $nowUtc.AddDays(-3).AddHours(2).ToString("o")
            is_follow_up = $false
        }
    )
    sessions = @(
        @{
            id = $testSessId
            appointment_id = $testAptId
            clinic_id = $adminClinicId
            patient_id = $patientId
            status = "COMPLETED"
            all_steps_completed = $true
            total_steps = 5
            completed_steps = 5
            started_at = $startedUtc
            completed_at = $completedUtc
            duration_minutes = 60.0
            follow_up_recommended = $true
        }
    )
} | ConvertTo-Json -Depth 5

try {
    $reconRes = Invoke-RestMethod -Uri "$BaseUrl/analytics/reconcile" -Method Post -Headers $adminHeaders -Body $reconcileBody -ContentType "application/json"
    Assert-Step "3.1 Operational reconciliation commits appointments and sessions" ($reconRes.data.status -eq "RECONCILED" -and $reconRes.data.appointments_processed -eq 3)
} catch {
    Assert-Step "3.1 Operational reconciliation" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 4. PATIENT FEEDBACK & DUPLICATE ENFORCEMENT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 4] Patient Feedback & Duplicate Prevention ---" -ForegroundColor Yellow

$createdFeedbackId = $null
$feedbackBody = @{
    appointment_id = $testAptId
    session_id = $testSessId
    clinic_id = $adminClinicId
    doctor_id = $doctorId
    therapist_id = "t0000000-0000-0000-0000-000000000001"
    overall_rating = 5
    therapist_rating = 5
    facility_cleanliness_rating = 4
    comments = "Deeply authentic Abhyanga therapy; felt rejuvenated."
    is_anonymous = $true
} | ConvertTo-Json

try {
    $fbRes = Invoke-RestMethod -Uri "$BaseUrl/feedback" -Method Post -Headers $patientHeaders -Body $feedbackBody -ContentType "application/json"
    $createdFeedbackId = $fbRes.data.id
    Assert-Step "4.1 Eligible patient submits feedback (201 Created)" ($createdFeedbackId -ne $null -and $fbRes.data.status -eq "SUBMITTED")
} catch {
    Assert-Step "4.1 Eligible patient feedback submission" $false $_.Exception.Message
}

# 4.2 Duplicate submission check -> Must return 409 Conflict
try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/feedback" -Method Post -Headers $patientHeaders -Body $feedbackBody -ContentType "application/json"
    Assert-Step "4.2 Duplicate feedback submission rejected with 409 Conflict" $false "Expected 409 Conflict"
} catch {
    $dupStatus = [int]$_.Exception.Response.StatusCode
    Assert-Step "4.2 Duplicate feedback submission rejected with 409 Conflict" ($dupStatus -eq 409) "Status: $dupStatus"
}

# 4.3 Role check: Therapist cannot submit feedback -> Must return 403 Forbidden
try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/feedback" -Method Post -Headers $therapistHeaders -Body $feedbackBody -ContentType "application/json"
    Assert-Step "4.3 Therapist role forbidden from submitting feedback (403)" $false "Expected 403 Forbidden"
} catch {
    $thStatus = [int]$_.Exception.Response.StatusCode
    Assert-Step "4.3 Therapist role forbidden from submitting feedback (403)" ($thStatus -eq 403) "Status: $thStatus"
}

# 4.4 Patient retrieves own feedback
try {
    $myFbRes = Invoke-RestMethod -Uri "$BaseUrl/feedback" -Method Get -Headers $patientHeaders
    $feedbacks = @($myFbRes.data.feedbacks)
    $found = $feedbacks | Where-Object { $_.appointment_id -eq $testAptId }
    Assert-Step "4.4 Patient retrieves their submitted feedback list" ($found -ne $null)
} catch {
    Assert-Step "4.4 Patient feedback list" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 5. ANONYMITY MASKING & STAFF REVIEW WORKFLOW
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 5] Anonymity Masking & Staff Acknowledgment ---" -ForegroundColor Yellow

if ($createdFeedbackId) {
    try {
        $adminFbRes = Invoke-RestMethod -Uri "$BaseUrl/feedback/$createdFeedbackId" -Method Get -Headers $adminHeaders
        # Anonymous feedback masks patient_id for staff
        Assert-Step "5.1 Anonymous feedback masks patient identity for clinic staff" ($adminFbRes.data.patient_id -eq "00000000-0000-0000-0000-000000000000")
    } catch {
        Assert-Step "5.1 Anonymity masking check" $false $_.Exception.Message
    }

    # Clinic admin acknowledges feedback
    try {
        $ackBody = @{
            staff_notes = "Thank you for the warm words; shared compliments with the therapy team."
            status = "ACKNOWLEDGED"
        } | ConvertTo-Json

        $ackRes = Invoke-RestMethod -Uri "$BaseUrl/feedback/$createdFeedbackId/acknowledge" -Method Post -Headers $adminHeaders -Body $ackBody -ContentType "application/json"
        Assert-Step "5.2 Clinic staff acknowledges feedback (transitions to ACKNOWLEDGED)" ($ackRes.data.status -eq "ACKNOWLEDGED" -and $ackRes.data.acknowledged_at -ne $null)
    } catch {
        Assert-Step "5.2 Clinic staff acknowledgment" $false $_.Exception.Message
    }
}

# ------------------------------------------------------------------------------
# 6. ADVERSE EVENT REPORTING & ISSUE MANAGEMENT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 6] Adverse Event / Complication Reporting ---" -ForegroundColor Yellow

$adverseEventId = $null
$adverseBody = @{
    clinic_id = $adminClinicId
    patient_id = $patientId
    session_id = $testSessId
    severity = "HIGH"
    symptoms = "Hypotension and mild skin erythema observed during Snehan protocol"
    vitals_at_incident = @{ bp = "85/55"; pulse = 58 }
    immediate_actions_taken = "Patient rested in supine position, administered warm herbal decoction"
    attending_doctor_id = $doctorId
} | ConvertTo-Json

try {
    $aeRes = Invoke-RestMethod -Uri "$BaseUrl/feedback/adverse-events" -Method Post -Headers $docHeaders -Body $adverseBody -ContentType "application/json"
    $adverseEventId = $aeRes.data.id
    Assert-Step "6.1 Clinical adverse event reported (status REPORTED)" ($adverseEventId -ne $null -and $aeRes.data.status -eq "REPORTED")
} catch {
    Assert-Step "6.1 Adverse event reporting" $false $_.Exception.Message
}

if ($adverseEventId) {
    # Doctor reviews and resolves adverse event
    try {
        $resolveBody = @{
            staff_notes = "Patient vitals stabilized at BP 115/75. Approved discharge with post-care rest."
            status = "RESOLVED"
        } | ConvertTo-Json

        $resAe = Invoke-RestMethod -Uri "$BaseUrl/feedback/adverse-events/$adverseEventId/acknowledge" -Method Post -Headers $docHeaders -Body $resolveBody -ContentType "application/json"
        Assert-Step "6.2 Doctor resolves adverse event (status RESOLVED, resolved_at recorded)" ($resAe.data.status -eq "RESOLVED" -and $resAe.data.resolved_at -ne $null)
    } catch {
        Assert-Step "6.2 Doctor adverse event resolution" $false $_.Exception.Message
    }
}

# ------------------------------------------------------------------------------
# 7. CLINIC DASHBOARD METRICS CALCULATION
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 7] Clinic Dashboard Metrics Calculation ---" -ForegroundColor Yellow

$todayStr = (Get-Date).ToString("yyyy-MM-dd")
$monthAgoStr = (Get-Date).AddDays(-30).ToString("yyyy-MM-dd")

try {
    $dashRes = Invoke-RestMethod -Uri "$BaseUrl/analytics/clinics/$adminClinicId/dashboard?start_date=$monthAgoStr&end_date=$todayStr" -Method Get -Headers $adminHeaders
    $m = $dashRes.data

    Assert-Step "7.1 Booking Conversion calculated" ($m.booking_conversion.value -ge 0.0 -and $m.booking_conversion.unit -eq "%")
    Assert-Step "7.2 Cancellation Rate calculated" ($m.cancellation_rate.value -ge 0.0 -and $m.cancellation_rate.unit -eq "%")
    Assert-Step "7.3 No-Show Rate calculated" ($m.no_show_rate.value -ge 0.0 -and $m.no_show_rate.unit -eq "%")
    Assert-Step "7.4 Checklist Adherence calculated" ($m.checklist_adherence.value -ge 0.0 -and $m.checklist_adherence.unit -eq "%")
    Assert-Step "7.5 Average Session Duration in minutes" ($m.average_session_duration.value -ge 0.0 -and $m.average_session_duration.unit -eq "minutes")
    Assert-Step "7.6 Average Feedback Rating (stars)" ($m.average_feedback_rating.value -ge 0.0 -and $m.average_feedback_rating.unit -eq "stars")
    Assert-Step "7.7 Therapist Utilization calculated" ($m.therapist_utilization.value -ge 0.0 -and $m.therapist_utilization.unit -eq "%")
    Assert-Step "7.8 Follow-Up Completion rate calculated" ($m.follow_up_completion.value -ge 0.0 -and $m.follow_up_completion.unit -eq "%")
    Assert-Step "7.9 Complications breakdown and resolved rate" ($m.complications.total_count -ge 1 -and $m.complications.resolved_rate_pct -ge 0.0)
} catch {
    Assert-Step "7.1 Dashboard metrics query" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 8. EDGE CASES: EMPTY DENOMINATORS & DATE BOUNDARIES
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 8] Edge Cases & Date Boundaries ---" -ForegroundColor Yellow

# Empty Date Range in Future (Year 2099) -> Must handle zero denominators without 500 error
try {
    $emptyRes = Invoke-RestMethod -Uri "$BaseUrl/analytics/clinics/$adminClinicId/dashboard?start_date=2099-01-01&end_date=2099-01-31" -Method Get -Headers $adminHeaders
    $em = $emptyRes.data
    Assert-Step "8.1 Empty date range returns zero values without division-by-zero" ($em.booking_conversion.value -eq 0.0 -and $em.complications.total_count -eq 0)
} catch {
    Assert-Step "8.1 Empty date range" $false $_.Exception.Message
}

# Inverted Date Boundary (start_date > end_date) -> Must return 400 Bad Request
try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/analytics/clinics/$adminClinicId/dashboard?start_date=2026-12-31&end_date=2026-01-01" -Method Get -Headers $adminHeaders
    Assert-Step "8.2 Inverted date boundary returns 400 Bad Request" $false "Expected 400 Bad Request"
} catch {
    $invStatus = [int]$_.Exception.Response.StatusCode
    Assert-Step "8.2 Inverted date boundary returns 400 Bad Request" ($invStatus -eq 400) "Status: $invStatus"
}

# ------------------------------------------------------------------------------
# 9. MULTI-TENANCY & CROSS-CLINIC ACCESS DENIAL
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 9] Multi-Tenancy & Cross-Clinic Denial ---" -ForegroundColor Yellow

$otherClinicId = "99999999-9999-9999-9999-999999999999"
try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/analytics/clinics/$otherClinicId/dashboard" -Method Get -Headers $adminHeaders
    Assert-Step "9.1 Cross-clinic access denied with 403 Forbidden" $false "Expected 403 Forbidden"
} catch {
    $crossStatus = [int]$_.Exception.Response.StatusCode
    Assert-Step "9.1 Cross-clinic access denied with 403 Forbidden" ($crossStatus -eq 403) "Status: $crossStatus"
}

# ------------------------------------------------------------------------------
# 10. CSV EXPORT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 10] CSV Data Exports ---" -ForegroundColor Yellow

try {
    $csvRes = Invoke-WebRequest -Uri "$BaseUrl/analytics/clinics/$adminClinicId/export?type=metrics" -Method Get -Headers $adminHeaders -UseBasicParsing
    $contentType = $csvRes.Headers["Content-Type"]
    $hasHeader = $csvRes.Content -like "*Booking Conversion*"
    Assert-Step "10.1 Metrics CSV export returns text/csv with formatted metrics" ($contentType -like "*text/csv*" -and $hasHeader)
} catch {
    Assert-Step "10.1 Metrics CSV export" $false $_.Exception.Message
}

try {
    $fbCsv = Invoke-WebRequest -Uri "$BaseUrl/analytics/clinics/$adminClinicId/export?type=feedback" -Method Get -Headers $adminHeaders -UseBasicParsing
    $hasFbHeader = $fbCsv.Content -like "*Overall Rating*"
    Assert-Step "10.2 Feedback CSV export returns text/csv with feedback details" ($hasFbHeader)
} catch {
    Assert-Step "10.2 Feedback CSV export" $false $_.Exception.Message
}

try {
    $aeCsv = Invoke-WebRequest -Uri "$BaseUrl/analytics/clinics/$adminClinicId/export?type=adverse-events" -Method Get -Headers $adminHeaders -UseBasicParsing
    $hasAeHeader = $aeCsv.Content -like "*Severity*"
    Assert-Step "10.3 Adverse events CSV export returns text/csv with clinical incident records" ($hasAeHeader)
} catch {
    Assert-Step "10.3 Adverse events CSV export" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# SUMMARY & VERIFICATION
# ------------------------------------------------------------------------------
Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  TEST RESULTS SUMMARY" -ForegroundColor White
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  TOTAL PASSED: $PassedTests" -ForegroundColor Green
Write-Host "  TOTAL FAILED: $FailedTests" -ForegroundColor $(if ($FailedTests -eq 0) { "Green" } else { "Red" })
Write-Host "==========================================================================" -ForegroundColor Cyan

if ($FailedTests -gt 0) {
    exit 1
} else {
    exit 0
}
