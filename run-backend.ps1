# PanchakarmaFlow — Run All Backend Services
param(
    [switch]$SkipInfra,
    [switch]$Build,
    [switch]$Windows
)

$ErrorActionPreference = "Stop"

# Resolve Directory Paths
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
if (Test-Path (Join-Path $ScriptDir "backend")) {
    $RootDir = $ScriptDir
    $BackendDir = Join-Path $ScriptDir "backend"
} elseif (Test-Path (Join-Path $ScriptDir "cmd")) {
    $RootDir = Split-Path -Parent $ScriptDir
    $BackendDir = $ScriptDir
} else {
    $RootDir = $PWD.Path
    $BackendDir = Join-Path $RootDir "backend"
}

$BinDir = Join-Path $BackendDir "bin"
$LogsDir = Join-Path $BackendDir "logs"
$PidFile = Join-Path $BackendDir ".backend-pids.json"

if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }
if (-not (Test-Path $LogsDir)) { New-Item -ItemType Directory -Path $LogsDir -Force | Out-Null }

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor DarkGreen
Write-Host "   PANCHAKARMA FLOW -- ALL BACKEND SERVICES ORCHESTRATOR" -ForegroundColor Green
Write-Host "==========================================================================" -ForegroundColor DarkGreen
Write-Host ""

# 1. Docker Infrastructure Check
if (-not $SkipInfra) {
    Write-Host "[1/4] Checking local infrastructure (PostgreSQL, Redis, Kafka)..." -ForegroundColor Cyan
    try {
        $dockerCheck = docker info 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "      Docker is active. Starting containers..." -ForegroundColor Gray
            docker compose -f (Join-Path $RootDir "docker-compose.yml") up -d
            Write-Host "      Infrastructure ready." -ForegroundColor Green
        } else {
            Write-Host "      [Notice] Docker is not running. Services will run in standalone mode." -ForegroundColor Yellow
        }
    } catch {
        Write-Host "      [Notice] Docker check skipped." -ForegroundColor Yellow
    }
} else {
    Write-Host "[1/4] Skipping infrastructure check (-SkipInfra)." -ForegroundColor Gray
}

# 2. Service Definitions
$Services = @(
    @{ Name = "identity-service";           Cmd = "./cmd/identity-service";           Http = 8081; Grpc = 9081 },
    @{ Name = "clinic-service";             Cmd = "./cmd/clinic-service";             Http = 8082; Grpc = 9082 },
    @{ Name = "appointment-service";        Cmd = "./cmd/appointment-service";        Http = 8083; Grpc = 9083 },
    @{ Name = "therapy-service";            Cmd = "./cmd/therapy-service";            Http = 8084; Grpc = 9084 },
    @{ Name = "notification-service";       Cmd = "./cmd/notification-service";       Http = 8085; Grpc = 9085 },
    @{ Name = "feedback-analytics-service"; Cmd = "./cmd/feedback-analytics-service"; Http = 8086; Grpc = 9086 },
    @{ Name = "api-gateway";                Cmd = "./cmd/api-gateway";                Http = 8080; Grpc = 0 }
)

# 3. Port Cleanup
Write-Host ""
Write-Host "[2/4] Verifying ports (8080-8086)..." -ForegroundColor Cyan
foreach ($svc in $Services) {
    $port = $svc.Http
    $conns = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
    if ($conns) {
        foreach ($conn in $conns) {
            $procId = $conn.OwningProcess
            if ($procId -gt 0) {
                Write-Host "      Freeing port :$port (PID $procId)..." -ForegroundColor Gray
                Stop-Process -Id $procId -Force -ErrorAction SilentlyContinue
            }
        }
    }
}

# 4. Compilation
Write-Host ""
Write-Host "[3/4] Checking binaries in backend/bin/..." -ForegroundColor Cyan
Set-Location $BackendDir

$NeedCompile = $Build
if (-not $NeedCompile) {
    foreach ($svc in $Services) {
        $binPath = Join-Path $BinDir "$($svc.Name).exe"
        if (-not (Test-Path $binPath)) {
            $NeedCompile = $true
            break
        }
    }
}

if ($NeedCompile) {
    Write-Host "      Compiling 7 Go microservice binaries..." -ForegroundColor Gray
    foreach ($svc in $Services) {
        $binPath = Join-Path $BinDir "$($svc.Name).exe"
        Write-Host "      -> Compiling $($svc.Name)..." -NoNewline
        $buildOutput = go build -o $binPath $svc.Cmd 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host " FAILED" -ForegroundColor Red
            Write-Host $buildOutput -ForegroundColor Red
            exit 1
        }
        Write-Host " OK" -ForegroundColor Green
    }
} else {
    Write-Host "      Binaries ready." -ForegroundColor Green
}

# 5. Launch All Services
Write-Host ""
Write-Host "[4/4] Starting microservices..." -ForegroundColor Cyan
$Running = @()

foreach ($svc in $Services) {
    $binPath = Join-Path $BinDir "$($svc.Name).exe"
    $logOut = Join-Path $LogsDir "$($svc.Name).out.log"
    $logErr = Join-Path $LogsDir "$($svc.Name).err.log"

    if ($Windows) {
        $proc = Start-Process -FilePath $binPath -WorkingDirectory $BackendDir -PassThru
    } else {
        $proc = Start-Process -FilePath $binPath -WorkingDirectory $BackendDir -RedirectStandardOutput $logOut -RedirectStandardError $logErr -PassThru -WindowStyle Hidden
    }

    $Running += [PSCustomObject]@{
        Name = $svc.Name
        PID  = $proc.Id
        Http = $svc.Http
        Grpc = $svc.Grpc
        Log  = $logOut
    }

    $grpcInfo = if ($svc.Grpc -gt 0) { "gRPC :$($svc.Grpc)" } else { "          " }
    Write-Host ("      [ONLINE] {0,-26} -> HTTP :{1}  {2}  (PID: {3})" -f $svc.Name, $svc.Http, $grpcInfo, $proc.Id) -ForegroundColor Green

    Start-Sleep -Milliseconds 350
}

# Save PIDs
$Running | ConvertTo-Json | Set-Content $PidFile -Force

Write-Host ""
Write-Host "==========================================================================" -ForegroundColor DarkGreen
Write-Host "   ALL 7 BACKEND SERVICES ARE LIVE AND OPERATIONAL!" -ForegroundColor Green
Write-Host "==========================================================================" -ForegroundColor DarkGreen
Write-Host "   Unified Gateway : http://localhost:8080/api/v1" -ForegroundColor Yellow
Write-Host "   Service Logs    : $LogsDir" -ForegroundColor Gray
Write-Host "   PID File        : $PidFile" -ForegroundColor Gray
Write-Host ""
Write-Host "   To stop all services, run:  .\stop-backend.ps1" -ForegroundColor Cyan
Write-Host "   Or press Ctrl+C in this terminal." -ForegroundColor Gray
Write-Host "==========================================================================" -ForegroundColor DarkGreen
Write-Host ""

# Process Supervision Loop
try {
    while ($true) {
        Start-Sleep -Seconds 2
        foreach ($p in $Running) {
            $alive = Get-Process -Id $p.PID -ErrorAction SilentlyContinue
            if (-not $alive) {
                Write-Host "[WARNING] Service $($p.Name) (PID $($p.PID)) has exited. Check log: $($p.Log)" -ForegroundColor Red
            }
        }
    }
} finally {
    Write-Host ""
    Write-Host "==> Shutting down all PanchakarmaFlow backend services..." -ForegroundColor Yellow
    foreach ($p in $Running) {
        if ($p.PID) {
            Stop-Process -Id $p.PID -Force -ErrorAction SilentlyContinue
            Write-Host "    Stopped $($p.Name) (PID: $($p.PID))" -ForegroundColor Gray
        }
    }
    if (Test-Path $PidFile) { Remove-Item $PidFile -Force -ErrorAction SilentlyContinue }
    Write-Host "==> All backend services stopped cleanly." -ForegroundColor Green
}
