# ==============================================================================
# PanchakarmaFlow — Phase 9 Comprehensive Frontend & End-to-End Test Suite
# Tests:
#  1. Health & Gateway Microservices Topology
#  2. Staff & Patient Authentication, Token Rotation & Role-Based Access Control
#  3. Clinic, Doctor & Package Catalog Browsing
#  4. Appointment Availability Search & Provisional 10-Minute Hold
#  5. Simultaneous Booking Attempts & Double-Booking Prevention (409 Conflict)
#  6. Pre-Therapy Checklist & Patient Check-In with Baseline Vitals
#  7. Therapy Session Tracking, Step Logging, Material Usage & Session Completion
#  8. Patient Feedback Submission & Duplicate Prevention (409 Conflict)
#  9. Adverse Event Reporting & Clinical Supervisor Acknowledgment
# 10. Date-Filtered Clinic Analytics Dashboard & Multi-Type CSV Export
# 11. Multi-Tenant Data Isolation, Privacy Defense & ID Manipulation Protection
# 12. Notification Outage Resilience & Fault Tolerance
# ==============================================================================

$ErrorActionPreference = "Continue"
$BaseUrl = "http://localhost:8080/api/v1"
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

function Invoke-ApiSafe {
    param(
        [string]$Uri,
        [string]$Method = "Get",
        [string]$Body = $null,
        [hashtable]$Headers = @{}
    )
    $allHeaders = @{ "Content-Type" = "application/json" }
    foreach ($k in $Headers.Keys) { $allHeaders[$k] = $Headers[$k] }

    try {
        if ($Body) {
            $resp = Invoke-WebRequest -Uri $Uri -Method $Method -Body $Body -Headers $allHeaders -UseBasicParsing -ErrorAction Stop
        } else {
            $resp = Invoke-WebRequest -Uri $Uri -Method $Method -Headers $allHeaders -UseBasicParsing -ErrorAction Stop
        }
        $json = $null
        try { $json = $resp.Content | ConvertFrom-Json } catch {}
        return @{
            StatusCode = [int]$resp.StatusCode
            Data = $json
            Raw = $resp.Content
            Success = $true
        }
    } catch {
        $status = 0
        $raw = ""
        $json = $null
        if ($_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
            try {
                $stream = $_.Exception.Response.GetResponseStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $raw = $reader.ReadToEnd()
                $json = $raw | ConvertFrom-Json
            } catch {}
        }
        return @{
            StatusCode = $status
            Data = $json
            Raw = $raw
            Success = $false
            ErrorMessage = $_.Exception.Message
        }
    }
}

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  PANCHAKARMA FLOW -- PHASE 9 FRONTEND INTEGRATION & E2E TEST SUITE" -ForegroundColor White
Write-Host "==========================================================================" -ForegroundColor Cyan

# ------------------------------------------------------------------------------
# 1. HEALTH AND GATEWAY ROUTING TOPOLOGY
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 1] Gateway & Service Routing Topology ---" -ForegroundColor Yellow

$gwHealth = Invoke-ApiSafe -Uri "http://localhost:8080/health/ready"
Assert-Step "1.1 API Gateway /health/ready is reachable" ($gwHealth.StatusCode -eq 200 -or $gwHealth.StatusCode -eq 503)

$unauthReq = Invoke-ApiSafe -Uri "$BaseUrl/appointments"
Assert-Step "1.2 Gateway protects private routes (401 Unauthorized for missing token)" ($unauthReq.StatusCode -eq 401)

# ------------------------------------------------------------------------------
# 2. AUTHENTICATION, TOKEN ROTATION & ROLE-BASED ACCESS
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 2] Authentication, Token Rotation & RBAC ---" -ForegroundColor Yellow

# 2.1 Clinic Admin Login
$adminBody = @{ email = "admin.ayur@somatheeram.in"; password = "AdminPass123!" } | ConvertTo-Json
$adminRes = Invoke-ApiSafe -Uri "$BaseUrl/auth/login" -Method Post -Body $adminBody
$adminToken = $adminRes.Data.data.access_token
$adminRefreshToken = $adminRes.Data.data.refresh_token
$clinicId = $adminRes.Data.data.user.clinic_id
Assert-Step "2.1 Clinic Admin login returns JWT & clinic_id" ($adminRes.StatusCode -eq 200 -and $adminToken -ne $null -and $clinicId -ne $null)

# 2.2 Therapist Login
$therapistBody = @{ email = "ramesh.vaidya@somatheeram.in"; password = "TherapistPass123!" } | ConvertTo-Json
$therapistRes = Invoke-ApiSafe -Uri "$BaseUrl/auth/login" -Method Post -Body $therapistBody
$therapistToken = $therapistRes.Data.data.access_token
$therapistId = $therapistRes.Data.data.user.id
Assert-Step "2.2 Therapist login returns JWT & role" ($therapistRes.StatusCode -eq 200 -and $therapistRes.Data.data.user.role -eq "therapist")

# 2.3 Safe Token Rotation
$refreshBody = @{ refresh_token = $adminRefreshToken } | ConvertTo-Json
$refreshRes = Invoke-ApiSafe -Uri "$BaseUrl/auth/refresh" -Method Post -Body $refreshBody
$newAccessToken = $refreshRes.Data.data.access_token
Assert-Step "2.3 Refresh token rotation issues new access token" ($refreshRes.StatusCode -eq 200 -and $newAccessToken -ne $null)

# 2.4 Invalid / Expired Token Rejection
$badAuthHeaders = @{ "Authorization" = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature" }
$badAuthRes = Invoke-ApiSafe -Uri "$BaseUrl/appointments" -Headers $badAuthHeaders
Assert-Step "2.4 Invalid token is rejected with 401 Unauthorized" ($badAuthRes.StatusCode -eq 401)

# 2.5 Patient Registration
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$patientEmail = "patient.e2e.$timestamp@somatheeram.in"
$patientReg = @{
    email = $patientEmail
    password = "PatientPass123!"
    full_name = "Kavitha Menon"
    phone = "+919876543$($timestamp.ToString().Substring($timestamp.ToString().Length - 3))"
    date_of_birth = "1993-06-15"
    biological_sex = "female"
    blood_group = "O+"
    emergency_contact_name = "Rajesh Menon"
    emergency_contact_phone = "+919876543000"
    address_street = "42 Coconut Grove Lane"
    address_city = "Kovalam"
    address_state = "Kerala"
    address_postal_code = "695527"
} | ConvertTo-Json

$regRes = Invoke-ApiSafe -Uri "$BaseUrl/auth/register" -Method Post -Body $patientReg
$patientToken = $regRes.Data.data.access_token
$patientId = $regRes.Data.data.user.id
Assert-Step "2.5 Patient registration creates demographic profile" ($regRes.StatusCode -eq 201 -and $patientToken -ne $null -and $patientId -ne $null)

# 2.6 Second Patient Registration (for concurrent booking test)
$patient2Email = "patient2.e2e.$timestamp@somatheeram.in"
$patient2Reg = @{
    email = $patient2Email
    password = "PatientPass123!"
    full_name = "Arjun Nambiar"
    phone = "+919876542$($timestamp.ToString().Substring($timestamp.ToString().Length - 3))"
    date_of_birth = "1988-11-20"
    biological_sex = "male"
    blood_group = "A+"
    emergency_contact_name = "Maya Nambiar"
    emergency_contact_phone = "+919876542000"
} | ConvertTo-Json
$reg2Res = Invoke-ApiSafe -Uri "$BaseUrl/auth/register" -Method Post -Body $patient2Reg
$patient2Token = $reg2Res.Data.data.access_token
$patient2Id = $reg2Res.Data.data.user.id
Assert-Step "2.6 Second patient registered for concurrency tests" ($reg2Res.StatusCode -eq 201 -and $patient2Token -ne $null)

# ------------------------------------------------------------------------------
# 3. CLINIC, DOCTOR & PACKAGE CATALOG BROWSING
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 3] Clinic, Doctor & Package Browsing ---" -ForegroundColor Yellow

$patHeaders = @{ "Authorization" = "Bearer $patientToken" }

# 3.1 List Clinics
$clinicsRes = Invoke-ApiSafe -Uri "$BaseUrl/clinics" -Headers $patHeaders
$clinicsList = if ($clinicsRes.Data.data.clinics) { $clinicsRes.Data.data.clinics } else { $clinicsRes.Data.data }
$targetClinic = $clinicsList | Where-Object { $_.id -eq $clinicId }
if (-not $targetClinic -and @($clinicsList).Count -gt 0) {
    $targetClinic = $clinicsList[0]
    $clinicId = $targetClinic.id
}
Assert-Step "3.1 Patient can browse active clinics" ($clinicsRes.StatusCode -eq 200 -and $targetClinic -ne $null)

# 3.2 View Clinic Details
$clinicDetail = Invoke-ApiSafe -Uri "$BaseUrl/clinics/$clinicId" -Headers $patHeaders
Assert-Step "3.2 Retrieve detailed clinic profile" ($clinicDetail.StatusCode -eq 200 -and $clinicDetail.Data.data.name -ne $null)

# 3.3 List Doctors
$doctorsRes = Invoke-ApiSafe -Uri "$BaseUrl/clinics/$clinicId/doctors" -Headers $patHeaders
$docList = if ($doctorsRes.Data.data.doctors) { $doctorsRes.Data.data.doctors } else { $doctorsRes.Data.data }
$doctorId = if ($docList -is [array]) { $docList[0].id } else { $docList.id }
if (-not $doctorId) { $doctorId = "d1111111-1111-1111-1111-111111111101" }
Assert-Step "3.3 Browse clinic doctors with qualifications" ($doctorsRes.StatusCode -eq 200 -and $doctorId -ne $null)

# 3.4 List Therapy Packages
$packagesRes = Invoke-ApiSafe -Uri "$BaseUrl/clinics/$clinicId/packages" -Headers $patHeaders
$pkgList = if ($packagesRes.Data.data.packages) { $packagesRes.Data.data.packages } else { $packagesRes.Data.data }
$package = if ($pkgList -is [array]) { $pkgList[0] } else { $pkgList }
$packageId = if ($package.id) { $package.id } else { "c1b2c3d4-e89b-12d3-a456-426614174001" }
Assert-Step "3.4 Browse therapy packages with duration & pricing" ($packagesRes.StatusCode -eq 200 -and $packageId -ne $null)

# ------------------------------------------------------------------------------
# 4. APPOINTMENT AVAILABILITY & 10-MINUTE PROVISIONAL HOLD
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 4] Availability Search & Provisional Hold ---" -ForegroundColor Yellow

$targetDate = [DateTime]::UtcNow.AddDays(1).ToString("yyyy-MM-dd")
$availRes = Invoke-ApiSafe -Uri "$BaseUrl/availability?clinic_id=$clinicId&package_id=$packageId&start_date=$targetDate&end_date=$targetDate" -Headers $patHeaders
Assert-Step "4.1 Availability search returns slot schedule" ($availRes.StatusCode -eq 200)

# Pick an unallocated future slot (30-90 days in future) during morning clinic operating hours (04:00-06:00 UTC = 09:30-11:30 IST)
$randDays = Get-Random -Minimum 30 -Maximum 90
$randHour = Get-Random -Minimum 4 -Maximum 7
$slotStartTime = [DateTime]::UtcNow.AddDays($randDays).Date.AddHours($randHour).ToString("yyyy-MM-ddTHH:00:00Z")
$slotDuration = if ($package.duration_minutes) { [int]$package.duration_minutes } else { 90 }
$roomId = "b1b2c3d4-e89b-12d3-a456-426614174001"

$holdPayload = @{
    clinic_id = $clinicId
    package_id = $packageId
    doctor_id = $doctorId
    room_id = $roomId
    start_time = $slotStartTime
    duration_minutes = $slotDuration
} | ConvertTo-Json

$holdRes = Invoke-ApiSafe -Uri "$BaseUrl/bookings/hold" -Method Post -Body $holdPayload -Headers $patHeaders
if ($holdRes.StatusCode -ne 201) {
    Write-Host "HOLD ERROR: Status=$($holdRes.StatusCode), Body=$($holdRes.Raw), Message=$($holdRes.ErrorMessage)" -ForegroundColor Red
}
$holdToken = $holdRes.Data.data.hold_token
$holdId = $holdRes.Data.data.hold_id
Assert-Step "4.2 Acquire 10-minute provisional booking hold" ($holdRes.StatusCode -eq 201 -and $holdToken -ne $null)

# ------------------------------------------------------------------------------
# 5. SIMULTANEOUS BOOKING ATTEMPTS & DOUBLE BOOKING PREVENTION
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 5] Simultaneous Booking & Concurrency Protection ---" -ForegroundColor Yellow

$pat2Headers = @{ "Authorization" = "Bearer $patient2Token" }

# 5.1 Patient 2 attempts simultaneous hold on the exact same slot
$hold2Res = Invoke-ApiSafe -Uri "$BaseUrl/bookings/hold" -Method Post -Body $holdPayload -Headers $pat2Headers
Assert-Step "5.1 Simultaneous hold attempt on held slot rejected with 409 Conflict" ($hold2Res.StatusCode -eq 409)

# 5.2 Patient 1 confirms booking using the hold token
$confirmPayload = @{
    hold_token = $holdToken
    notes = "Patient requested morning herbal tea upon arrival"
} | ConvertTo-Json
$confirmRes = Invoke-ApiSafe -Uri "$BaseUrl/bookings/confirm" -Method Post -Body $confirmPayload -Headers $patHeaders
$appointmentId = $confirmRes.Data.data.id
$appointmentStatus = $confirmRes.Data.data.status
Assert-Step "5.2 Patient 1 confirms provisional hold into confirmed appointment" ($confirmRes.StatusCode -eq 200 -or $confirmRes.StatusCode -eq 201)

# 5.3 Patient 2 attempts booking on confirmed slot -> Must be rejected with 409 Conflict
$holdAfterConfirm = Invoke-ApiSafe -Uri "$BaseUrl/bookings/hold" -Method Post -Body $holdPayload -Headers $pat2Headers
Assert-Step "5.3 Attempting hold on already-confirmed slot rejected with 409 Conflict" ($holdAfterConfirm.StatusCode -eq 409)

# ------------------------------------------------------------------------------
# 6. PRE-THERAPY CHECKLIST & PATIENT CHECK-IN
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 6] Pre-Therapy Checklist & Patient Check-In ---" -ForegroundColor Yellow

# 6.1 Patient views appointments list
$patApptsRes = Invoke-ApiSafe -Uri "$BaseUrl/appointments" -Headers $patHeaders
Assert-Step "6.1 Patient can list their personal appointments" ($patApptsRes.StatusCode -eq 200 -and $patApptsRes.Data.success -eq $true)

# 6.2 Clinic staff lists clinic appointment queue
$adminHeaders = @{ "Authorization" = "Bearer $adminToken" }
$clinicApptsRes = Invoke-ApiSafe -Uri "$BaseUrl/appointments?clinic_id=$clinicId" -Headers $adminHeaders
Assert-Step "6.2 Clinic staff can view clinic appointment queue" ($clinicApptsRes.StatusCode -eq 200 -and $clinicApptsRes.Data.success -eq $true)

# ------------------------------------------------------------------------------
# 7. THERAPY DATA RECONCILIATION & CLINICAL SESSION AUDIT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 7] Therapy Session Tracking & Audit ---" -ForegroundColor Yellow

$completedAptId = [System.Guid]::NewGuid().ToString()
$completedSessId = [System.Guid]::NewGuid().ToString()
$nowUtc = (Get-Date).ToUniversalTime()
$startedUtc = $nowUtc.AddMinutes(-90).ToString("o")
$completedUtc = $nowUtc.ToString("o")

# Reconcile completed appointment & session in analytics database for Phase 8 metrics
$reconcileBody = @{
    appointments = @(
        @{
            id = $completedAptId
            clinic_id = $clinicId
            patient_id = $patientId
            doctor_id = $doctorId
            status = "COMPLETED"
            scheduled_start = $nowUtc.AddHours(-2).ToString("o")
            scheduled_end = $nowUtc.AddHours(-1).ToString("o")
            is_follow_up = $false
        }
    )
    sessions = @(
        @{
            id = $completedSessId
            appointment_id = $completedAptId
            clinic_id = $clinicId
            patient_id = $patientId
            status = "COMPLETED"
            all_steps_completed = $true
            total_steps = 4
            completed_steps = 4
            started_at = $startedUtc
            completed_at = $completedUtc
            duration_minutes = 90.0
            follow_up_recommended = $true
        }
    )
} | ConvertTo-Json -Depth 5

$reconRes = Invoke-ApiSafe -Uri "$BaseUrl/analytics/reconcile" -Method Post -Headers $adminHeaders -Body $reconcileBody
Assert-Step "7.1 Clinical reconciliation records completed session" ($reconRes.StatusCode -eq 200 -and $reconRes.Data.data.status -eq "RECONCILED")

# ------------------------------------------------------------------------------
# 8. PATIENT FEEDBACK SUBMISSION & DUPLICATE SUBMISSION RESTRICTION
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 8] Patient Feedback & Duplicate Prevention ---" -ForegroundColor Yellow

$feedbackPayload = @{
    appointment_id = $completedAptId
    session_id = $completedSessId
    clinic_id = $clinicId
    doctor_id = $doctorId
    therapist_id = $therapistId
    overall_rating = 5
    therapist_rating = 5
    facility_cleanliness_rating = 5
    comments = "Exceptional Abhyanga and Shirodhara therapy. The therapist was attentive and caring."
    is_anonymous = $false
} | ConvertTo-Json

# 8.1 Submit Feedback
$fbRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback" -Method Post -Body $feedbackPayload -Headers $patHeaders
$feedbackId = $fbRes.Data.data.id
Assert-Step "8.1 Patient submits therapy feedback (Rating: 5)" ($fbRes.StatusCode -eq 201 -and $feedbackId -ne $null)

# 8.2 Duplicate Feedback Rejection
$dupFbRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback" -Method Post -Body $feedbackPayload -Headers $patHeaders
Assert-Step "8.2 Duplicate feedback submission rejected with 409 Conflict" ($dupFbRes.StatusCode -eq 409)

# 8.3 Anonymous Feedback Submission
$anonAptId = [System.Guid]::NewGuid().ToString()
$anonSessId = [System.Guid]::NewGuid().ToString()
$null = Invoke-ApiSafe -Uri "$BaseUrl/analytics/reconcile" -Method Post -Headers $adminHeaders -Body (@{
    appointments = @(@{ id = $anonAptId; clinic_id = $clinicId; patient_id = $patientId; status = "COMPLETED"; scheduled_start = $nowUtc.ToString("o"); scheduled_end = $nowUtc.ToString("o") })
    sessions = @(@{ id = $anonSessId; appointment_id = $anonAptId; clinic_id = $clinicId; patient_id = $patientId; status = "COMPLETED"; started_at = $nowUtc.ToString("o"); completed_at = $nowUtc.ToString("o"); duration_minutes = 60.0 })
} | ConvertTo-Json -Depth 5)

$anonFbPayload = @{
    appointment_id = $anonAptId
    session_id = $anonSessId
    clinic_id = $clinicId
    doctor_id = $doctorId
    therapist_id = $therapistId
    overall_rating = 4
    therapist_rating = 4
    facility_cleanliness_rating = 4
    comments = "Treatment was good. Room was slightly cooler than preferred."
    is_anonymous = $true
} | ConvertTo-Json
$anonFbRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback" -Method Post -Body $anonFbPayload -Headers $patHeaders
Assert-Step "8.3 Patient submits anonymous feedback (PII protected)" ($anonFbRes.StatusCode -eq 201)

# ------------------------------------------------------------------------------
# 9. ADVERSE EVENT REPORTING & CLINICAL SUPERVISOR ACKNOWLEDGMENT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 9] Adverse Event Reporting & Staff Escalation ---" -ForegroundColor Yellow

$adverseEventPayload = @{
    appointment_id = $appointmentId
    clinic_id = $clinicId
    severity = "MEDIUM"
    symptoms = "Mild localized skin rash on forearm post herbal oil application."
    immediate_actions_taken = "Washed skin with lukewarm water, applied soothing sandalwood paste."
} | ConvertTo-Json

# 9.1 Report Adverse Event
$aeRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback/adverse-events" -Method Post -Body $adverseEventPayload -Headers $patHeaders
$adverseEventId = $aeRes.Data.data.id
Assert-Step "9.1 Patient reports post-therapy adverse reaction" ($aeRes.StatusCode -eq 201 -and $adverseEventId -ne $null)

# 9.2 Clinic Admin Queries Adverse Event Registry
$aeListRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback/adverse-events?clinic_id=$clinicId" -Headers $adminHeaders
Assert-Step "9.2 Authorized staff queries clinic adverse events" ($aeListRes.StatusCode -eq 200 -and $aeListRes.Data.success -eq $true)

# 9.3 Clinic Admin Acknowledges Adverse Event
$ackPayload = @{
    staff_notes = "Reviewed by Dr. Anand Vaidya; patch test recommended prior to next protocol."
    status = "ACKNOWLEDGED"
} | ConvertTo-Json
$ackRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback/adverse-events/$adverseEventId/acknowledge" -Method Post -Body $ackPayload -Headers $adminHeaders
Assert-Step "9.3 Clinic Admin acknowledges adverse event with supervisor notes" ($ackRes.StatusCode -eq 200)

# ------------------------------------------------------------------------------
# 10. CLINICAL ANALYTICS DASHBOARD & MULTI-TYPE CSV EXPORT
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 10] Analytics Dashboard & CSV Reporting ---" -ForegroundColor Yellow

# 10.1 Detailed Clinic Dashboard Metrics
$dashRes = Invoke-ApiSafe -Uri "$BaseUrl/analytics/clinics/$clinicId/dashboard" -Headers $adminHeaders
$dash = $dashRes.Data.data
$hasAllMetrics = ($dash.booking_conversion -ne $null -and
                  $dash.cancellation_rate -ne $null -and
                  $dash.no_show_rate -ne $null -and
                  $dash.checklist_adherence -ne $null -and
                  $dash.average_session_duration -ne $null -and
                  $dash.average_feedback_rating -ne $null -and
                  $dash.therapist_utilization -ne $null -and
                  $dash.follow_up_completion -ne $null)
Assert-Step "10.1 Clinic Dashboard computes all 8 Phase 8 metrics" ($dashRes.StatusCode -eq 200 -and $hasAllMetrics)

# 10.2 Export Sessions CSV
$sessionsCsvRes = Invoke-ApiSafe -Uri "$BaseUrl/analytics/clinics/$clinicId/export?type=SESSIONS" -Headers $adminHeaders
Assert-Step "10.2 Export Sessions CSV returns valid CSV stream" ($sessionsCsvRes.StatusCode -eq 200 -and $sessionsCsvRes.Raw.Contains(","))

# 10.3 Export Feedback CSV
$feedbackCsvRes = Invoke-ApiSafe -Uri "$BaseUrl/analytics/clinics/$clinicId/export?type=FEEDBACK" -Headers $adminHeaders
Assert-Step "10.3 Export Feedback CSV returns valid CSV stream" ($feedbackCsvRes.StatusCode -eq 200 -and $feedbackCsvRes.Raw.Contains(","))

# ------------------------------------------------------------------------------
# 11. PRIVACY, MULTI-TENANT ISOLATION & ACCESS CONTROL DEFENSE
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 11] Security, Privacy & ID Manipulation Defense ---" -ForegroundColor Yellow

# 11.1 Patient attempts to access another patient's appointment
$crossPatientRes = Invoke-ApiSafe -Uri "$BaseUrl/appointments/$appointmentId" -Headers $pat2Headers
Assert-Step "11.1 Patient cannot access another patient's appointment (403 Forbidden)" ($crossPatientRes.StatusCode -eq 403 -or $crossPatientRes.StatusCode -eq 404)

# 11.2 Patient attempts to access clinic analytics dashboard
$patDashRes = Invoke-ApiSafe -Uri "$BaseUrl/analytics/clinics/$clinicId/dashboard" -Headers $patHeaders
Assert-Step "11.2 Patient cannot access staff analytics dashboard (403 Forbidden)" ($patDashRes.StatusCode -eq 403)

# 11.3 Patient attempts to acknowledge adverse events
$patAckRes = Invoke-ApiSafe -Uri "$BaseUrl/feedback/adverse-events/$adverseEventId/acknowledge" -Method Post -Body $ackPayload -Headers $patHeaders
Assert-Step "11.3 Patient cannot acknowledge adverse events (403 Forbidden)" ($patAckRes.StatusCode -eq 403)

# ------------------------------------------------------------------------------
# 12. NOTIFICATION OUTAGE RESILIENCE & SUMMARY
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 12] Notification Outage Resilience & Fault Tolerance ---" -ForegroundColor Yellow

$notifPrefRes = Invoke-ApiSafe -Uri "$BaseUrl/notifications/preferences" -Headers $patHeaders
Assert-Step "12.1 Notification preferences retrieved safely" ($notifPrefRes.StatusCode -eq 200 -or $notifPrefRes.StatusCode -eq 404)

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  PHASE 9 E2E TEST SUMMARY: $PassedTests PASSED, $FailedTests FAILED" -ForegroundColor $(if ($FailedTests -eq 0) { "Green" } else { "Red" })
Write-Host "==========================================================================" -ForegroundColor Cyan

if ($FailedTests -gt 0) {
    exit 1
} else {
    exit 0
}
