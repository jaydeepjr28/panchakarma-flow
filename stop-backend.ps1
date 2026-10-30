<#
.SYNOPSIS
    Stops all running PanchakarmaFlow Go backend microservices and clears occupied ports.
#>

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
if (Test-Path (Join-Path $ScriptDir "backend")) {
    $BackendDir = Join-Path $ScriptDir "backend"
} else {
    $BackendDir = $ScriptDir
}

$PidFile = Join-Path $BackendDir ".backend-pids.json"

Write-Host "==> Stopping PanchakarmaFlow backend microservices..." -ForegroundColor Yellow

$StoppedAny = $false

# 1. Stop by recorded PIDs
if (Test-Path $PidFile) {
    try {
        $processes = Get-Content $PidFile -Raw | ConvertFrom-Json
        foreach ($proc in $processes) {
            if ($proc.PID) {
                $p = Get-Process -Id $proc.PID -ErrorAction SilentlyContinue
                if ($p) {
                    Stop-Process -Id $proc.PID -Force -ErrorAction SilentlyContinue
                    Write-Host "    Stopped $($proc.Name) (PID: $($proc.PID))" -ForegroundColor Gray
                    $StoppedAny = $true
                }
            }
        }
    } catch {
        # ignore parse error
    }
    Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
}

# 2. Safety check: ensure ports 8080 through 8086 are freed
$Ports = 8080..8086
foreach ($port in $Ports) {
    $conns = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
    if ($conns) {
        foreach ($conn in $conns) {
            if ($conn.OwningProcess -gt 0) {
                Write-Host "    Terminating leftover process on port :$port (PID: $($conn.OwningProcess))" -ForegroundColor Gray
                Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
                $StoppedAny = $true
            }
        }
    }
}

if ($StoppedAny) {
    Write-Host "==> All backend services have been stopped successfully." -ForegroundColor Green
} else {
    Write-Host "==> No active backend service processes were found." -ForegroundColor DarkGray
}
