# ==============================================================================
# PanchakarmaFlow — Comprehensive Phase 6.5 End-to-End Integration Test Suite
# Tests: Auth, RBAC, Admin Config, Direct Booking, PostgreSQL Double-Booking Protection,
#        Therapist Check-In, Step Lifecycle, Session Completion, Patient Privacy
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

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host "  PANCHAKARMA FLOW -- PHASE 6.5 END-TO-END INTEGRATION TEST SUITE" -ForegroundColor White
Write-Host "==========================================================================" -ForegroundColor Cyan

# ------------------------------------------------------------------------------
# 1. AUTHENTICATION & ROLE-BASED ACCESS
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 1] Authentication & Access Control ---" -ForegroundColor Yellow

# 1.1 Clinic Admin Login
$adminBody = @{ email = "admin.ayur@somatheeram.in"; password = "AdminPass123!" } | ConvertTo-Json
try {
    $adminRes = Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body $adminBody -ContentType "application/json"
    $adminToken = $adminRes.data.access_token
    $clinicId = $adminRes.data.user.clinic_id
    Assert-Step "1.1 Clinic Admin login returns JWT" ($adminToken -ne $null -and $adminRes.data.user.role -eq "clinic_admin")
} catch {
    Assert-Step "1.1 Clinic Admin login" $false $_.Exception.Message
}

# 1.2 Therapist Login
$therapistBody = @{ email = "ramesh.vaidya@somatheeram.in"; password = "TherapistPass123!" } | ConvertTo-Json
try {
    $therapistRes = Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body $therapistBody -ContentType "application/json"
    $therapistToken = $therapistRes.data.access_token
    $therapistId = $therapistRes.data.user.id
    Assert-Step "1.2 Therapist login returns JWT" ($therapistToken -ne $null -and $therapistRes.data.user.role -eq "therapist")
} catch {
    Assert-Step "1.2 Therapist login" $false $_.Exception.Message
}

# 1.3 Unregistered User Rejection
$badLogin = @{ email = "unregistered.ghost@example.com"; password = "WrongPassword99!" } | ConvertTo-Json
$rejected = $false
try {
    Invoke-RestMethod -Uri "$BaseUrl/auth/login" -Method Post -Body $badLogin -ContentType "application/json"
} catch {
    $rejected = ([int]$_.Exception.Response.StatusCode -eq 401)
}
Assert-Step "1.3 Unregistered login returns 401 Unauthorized" $rejected

# 1.4 Patient Self-Registration
$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$patientEmail = "patient.e2e.$stamp@example.com"
$patientRegisterBody = @{
    email = $patientEmail
    password = "PatientPass123!"
    full_name = "Kavita Ramachandran"
    phone = "+919876599$($stamp.ToString().Substring($stamp.ToString().Length - 4))"
    date_of_birth = "1994-04-12"
    biological_sex = "female"
    emergency_contact_name = "Ramachandran Pillai"
    emergency_contact_phone = "+919876599000"
} | ConvertTo-Json

try {
    $regRes = Invoke-RestMethod -Uri "$BaseUrl/auth/register" -Method Post -Body $patientRegisterBody -ContentType "application/json"
    $patientToken = $regRes.data.access_token
    $patientId = $regRes.data.user.id
    Assert-Step "1.4 Patient registration creates user & demographic profile" ($patientToken -ne $null -and $patientId -ne $null)
} catch {
    Assert-Step "1.4 Patient registration" $false $_.Exception.Message
}

# 1.5 Privilege Escalation Defense (Patient cannot call Admin endpoints)
$forbidden = $false
try {
    $roomPayload = @{ name = "Hacked Room"; room_type = "steam_and_massage"; sanitation_buffer_minutes = 15 } | ConvertTo-Json
    Invoke-RestMethod -Uri "$BaseUrl/clinics/$clinicId/rooms" -Method Post -Body $roomPayload -Headers @{ Authorization = "Bearer $patientToken" } -ContentType "application/json"
} catch {
    $forbidden = ([int]$_.Exception.Response.StatusCode -eq 403)
}
Assert-Step "1.5 RBAC enforces 403 Forbidden when patient calls admin endpoint" $forbidden

# ------------------------------------------------------------------------------
# 2. CLINIC CATALOG & CONFIGURATION
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 2] Clinic Discovery & Catalog Management ---" -ForegroundColor Yellow

# 2.1 List Clinics
try {
    $clinicsRes = Invoke-RestMethod -Uri "$BaseUrl/clinics" -Method Get
    $clinics = $clinicsRes.data.clinics
    Assert-Step "2.1 Public clinics list retrieves active clinics" ($clinics.Count -ge 1)
} catch {
    Assert-Step "2.1 Public clinics list" $false $_.Exception.Message
}

# 2.2 List Doctors for Clinic
try {
    $doctorsRes = Invoke-RestMethod -Uri "$BaseUrl/clinics/$clinicId/doctors" -Method Get
    $doctors = $doctorsRes.data.doctors
    $doctorId = $doctors[0].id
    Assert-Step "2.2 Retrieve doctors for clinic" ($doctors.Count -ge 1 -and $doctorId -ne $null)
} catch {
    Assert-Step "2.2 Retrieve doctors for clinic" $false $_.Exception.Message
}

# 2.3 List Rooms for Clinic
try {
    $roomsRes = Invoke-RestMethod -Uri "$BaseUrl/clinics/$clinicId/rooms" -Method Get -Headers @{ Authorization = "Bearer $adminToken" }
    $rooms = $roomsRes.data.rooms
    $roomId = $rooms[0].id
    Assert-Step "2.3 Retrieve treatment rooms for clinic" ($rooms.Count -ge 1 -and $roomId -ne $null)
} catch {
    Assert-Step "2.3 Retrieve treatment rooms for clinic" $false $_.Exception.Message
}

# 2.4 List Packages for Clinic
try {
    $packagesRes = Invoke-RestMethod -Uri "$BaseUrl/clinics/$clinicId/packages" -Method Get
    $packages = $packagesRes.data.packages
    $packageId = $packages[0].id
    Assert-Step "2.4 Retrieve published therapy packages" ($packages.Count -ge 1 -and $packageId -ne $null)
} catch {
    Assert-Step "2.4 Retrieve published therapy packages" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 3. APPOINTMENT BOOKING & CONFLICT HANDLING
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 3] Appointment Booking & Double-Booking Protection ---" -ForegroundColor Yellow

# Schedule a unique day in the future within doctor working hours (09:00 AM UTC)
$dayOffset = 3 + ($stamp % 25)
$targetDate = [DateTime]::UtcNow.AddDays($dayOffset).Date.AddHours(9)
$startTimeIso = $targetDate.ToString("yyyy-MM-ddTHH:mm:ssZ")

# 3.1 Patient Direct Booking
$bookingBody = @{
    clinic_id = $clinicId
    doctor_id = $doctorId
    room_id = $roomId
    package_id = $packageId
    start_time = $startTimeIso
    duration_minutes = 90
    notes = "Classical Abhyanga detox request"
} | ConvertTo-Json

try {
    $bookingRes = Invoke-RestMethod -Uri "$BaseUrl/bookings/direct" -Method Post -Body $bookingBody -Headers @{ Authorization = "Bearer $patientToken" } -ContentType "application/json"
    $appointmentId = $bookingRes.data.id
    Assert-Step "3.1 Patient successfully creates confirmed direct booking" ($appointmentId -ne $null -and $bookingRes.data.status -eq "CONFIRMED")
} catch {
    Assert-Step "3.1 Patient direct booking" $false $_.Exception.Message
}

# 3.2 Double-Booking Rejection (Concurrent doctor/room collision test)
$doubleBookRejected = $false
try {
    # Attempt second booking on the EXACT same doctor, room, and time slot
    $conflictBody = @{
        clinic_id = $clinicId
        doctor_id = $doctorId
        room_id = $roomId
        package_id = $packageId
        start_time = $startTimeIso
        duration_minutes = 90
        notes = "Conflicting concurrent booking attempt"
    } | ConvertTo-Json

    Invoke-RestMethod -Uri "$BaseUrl/bookings/direct" -Method Post -Body $conflictBody -Headers @{ Authorization = "Bearer $patientToken" } -ContentType "application/json"
} catch {
    $doubleBookRejected = ([int]$_.Exception.Response.StatusCode -eq 409)
}
Assert-Step "3.2 PostgreSQL exclusion constraint blocks overlapping slot (409 Conflict)" $doubleBookRejected

# 3.3 Patient Appointment Listing
try {
    $myApptsRes = Invoke-RestMethod -Uri "$BaseUrl/appointments?patient_id=$patientId" -Method Get -Headers @{ Authorization = "Bearer $patientToken" }
    $appts = $myApptsRes.data
    Assert-Step "3.3 Patient sees newly booked appointment in appointment history" ($appts.Count -eq 1 -and $appts[0].id -eq $appointmentId)
} catch {
    Assert-Step "3.3 Patient appointment listing" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 4. THERAPY EXECUTION WORKFLOW
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 4] Therapist Execution Lifecycle ---" -ForegroundColor Yellow

# 4.1 Therapist Patient Check-In
$checkInBody = @{
    appointment_id = $appointmentId
    clinic_id = $clinicId
    therapist_id = $therapistId
    baseline_vitals = @{
        blood_pressure = "120/80"
        pulse = 74
        temperature = 98.4
        respiratory_rate = 16
    }
    pre_checklist_verified = $true
    pre_checklist_data = @{
        fasting_verified = $true
        blood_pressure_normal = $true
        no_fever = $true
        consent_signed = $true
    }
} | ConvertTo-Json

try {
    $checkInRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/check-in" -Method Post -Body $checkInBody -Headers @{ Authorization = "Bearer $therapistToken" } -ContentType "application/json"
    $sessionId = $checkInRes.data.id
    $steps = $checkInRes.data.steps
    Assert-Step "4.1 Therapist check-in initializes session and therapy steps" ($sessionId -ne $null -and $steps.Count -ge 1)
} catch {
    Assert-Step "4.1 Therapist check-in" $false $_.Exception.Message
}

# 4.2 Start Therapy Session
try {
    $startRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/start" -Method Post -Headers @{ Authorization = "Bearer $therapistToken" } -Body "{}" -ContentType "application/json"
    Assert-Step "4.2 Start session transitions status to IN_PROGRESS" ($startRes.data.status -eq "IN_PROGRESS")
} catch {
    Assert-Step "4.2 Start session" $false $_.Exception.Message
}

# 4.3 Execute Step 1 (Start & Complete)
try {
    $step1Start = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/steps/1/start" -Method Post -Headers @{ Authorization = "Bearer $therapistToken" } -Body "{}" -ContentType "application/json"
    $step1End = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/steps/1/complete" -Method Post -Headers @{ Authorization = "Bearer $therapistToken" } -Body "{}" -ContentType "application/json"
    Assert-Step "4.3 Step 1 executes and completes successfully" ($step1End.data.status -eq "COMPLETED")
} catch {
    Assert-Step "4.3 Execute Step 1" $false $_.Exception.Message
}

# 4.4 Execute All Remaining Steps Sequentially
$remainingAllPassed = $true
$remainingMsg = ""
try {
    for ($i = 2; $i -le $steps.Count; $i++) {
        $stepStart = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/steps/$i/start" -Method Post -Headers @{ Authorization = "Bearer $therapistToken" } -Body "{}" -ContentType "application/json"
        $stepEnd = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/steps/$i/complete" -Method Post -Headers @{ Authorization = "Bearer $therapistToken" } -Body "{}" -ContentType "application/json"
        if ($stepEnd.data.status -ne "COMPLETED") {
            $remainingAllPassed = $false
            $remainingMsg = "Step $i did not return COMPLETED"
            break
        }
    }
} catch {
    $remainingAllPassed = $false
    $remainingMsg = $_.Exception.Message
}
Assert-Step "4.4 All subsequent therapy steps (2 to $($steps.Count)) execute and complete sequentially" $remainingAllPassed $remainingMsg

# 4.5 Record Materials Used
$materialBody = @{
    material_name = "Ksheerabala Thailam (Herbal Medicated Oil)"
    quantity = 150.0
    unit = "ml"
    batch_number = "KB-2026-088"
    notes = "Warmed to 38 degrees Celsius"
} | ConvertTo-Json
try {
    $matRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/materials" -Method Post -Body $materialBody -Headers @{ Authorization = "Bearer $therapistToken" } -ContentType "application/json"
    Assert-Step "4.5 Record clinical materials used during session" ($matRes.data.material_name -eq "Ksheerabala Thailam (Herbal Medicated Oil)")
} catch {
    Assert-Step "4.5 Record materials used" $false $_.Exception.Message
}

# 4.6 Complete Therapy Session
$completeBody = @{
    therapist_notes = "Patient responded well to Abhyanga and Svedana. Mild localized Pitta reduction noted."
    observations = "Normal perspiration, relaxed muscle tone."
    post_care_instructions = "Avoid cold showers for 4 hours; drink warm herbal water; avoid heavy food."
} | ConvertTo-Json
try {
    $compRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId/complete" -Method Post -Body $completeBody -Headers @{ Authorization = "Bearer $therapistToken" } -ContentType "application/json"
    Assert-Step "4.6 Session completed and marked COMPLETED" ($compRes.data.status -eq "COMPLETED")
} catch {
    Assert-Step "4.6 Complete session" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# 5. PATIENT PROGRESS & PRIVACY VERIFICATION
# ------------------------------------------------------------------------------
Write-Host "`n--- [TEST GROUP 5] Patient Progress & Privacy Verification ---" -ForegroundColor Yellow

# 5.1 Patient Views Session Details
try {
    $patientSessionRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId" -Method Get -Headers @{ Authorization = "Bearer $patientToken" }
    $patientView = $patientSessionRes.data
    $isCompleted = ($patientView.status -eq "COMPLETED")
    # Patient view must mask internal therapist clinical notes for HIPAA/clinical confidentiality
    $notesMasked = ($patientView.therapist_notes -eq $null -or $patientView.therapist_notes -eq "")
    $hasPostCare = ($patientView.post_care_instructions -ne $null -and $patientView.post_care_instructions.Length -gt 0)
    Assert-Step "5.1 Patient views completed session with confidential notes masked and post-care intact" ($isCompleted -and $notesMasked -and $hasPostCare)
} catch {
    Assert-Step "5.1 Patient views session" $false $_.Exception.Message
}

# 5.2 Therapist Views Same Session (Sees Full Clinical Notes)
try {
    $therapistSessionRes = Invoke-RestMethod -Uri "$BaseUrl/sessions/$sessionId" -Method Get -Headers @{ Authorization = "Bearer $therapistToken" }
    $therapistView = $therapistSessionRes.data
    $hasNotes = ($therapistView.therapist_notes -ne $null -and $therapistView.therapist_notes.Length -gt 0)
    Assert-Step "5.2 Authorized therapist retrieves unmasked clinical notes" $hasNotes
} catch {
    Assert-Step "5.2 Therapist views session" $false $_.Exception.Message
}

# ------------------------------------------------------------------------------
# SUMMARY
# ------------------------------------------------------------------------------
Write-Host ""
Write-Host "==========================================================================" -ForegroundColor Cyan
Write-Host ("  TOTAL TESTS: {0}  |  PASSED: {1}  |  FAILED: {2}" -f ($PassedTests + $FailedTests), $PassedTests, $FailedTests) -ForegroundColor White
Write-Host "==========================================================================" -ForegroundColor Cyan
if ($FailedTests -eq 0) {
    Write-Host "  >>> ALL PHASE 6.5 END-TO-END INTEGRATION SCENARIOS PASSED! <<<" -ForegroundColor Green
    exit 0
} else {
    Write-Host "  >>> SOME INTEGRATION CHECKS FAILED. SEE DETAILS ABOVE. <<<" -ForegroundColor Red
    exit 1
}
