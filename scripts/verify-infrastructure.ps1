<#
.SYNOPSIS
    Probes all NexusCommerce infrastructure services running in WSL2 from the host environment.

.DESCRIPTION
    Tests TCP connectivity and basic protocol ping against:
    - PostgreSQL (Port 15432)
    - Redis (Port 16379)
    - MongoDB (Port 27017)
    - MinIO S3 API (Port 9002)
    - ClickHouse HTTP Ping (Port 8123)
    - RabbitMQ Management API (Port 15672)
    - NATS HTTP Monitoring (Port 8222)
    - Elasticsearch Cluster Health (Port 9200)
#>

$ErrorActionPreference = "Continue"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Probing NexusCommerce Infrastructure Services" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

$services = @(
    @{ Name = "PostgreSQL"; Host = "localhost"; Port = 15432; Type = "TCP" },
    @{ Name = "Redis"; Host = "localhost"; Port = 16379; Type = "TCP" },
    @{ Name = "MongoDB"; Host = "localhost"; Port = 27017; Type = "TCP" },
    @{ Name = "MinIO API"; Host = "localhost"; Port = 9002; Type = "HTTP"; Url = "http://localhost:9002/minio/health/live" },
    @{ Name = "ClickHouse HTTP"; Host = "localhost"; Port = 8123; Type = "HTTP"; Url = "http://localhost:8123/ping" },
    @{ Name = "RabbitMQ Admin"; Host = "localhost"; Port = 15672; Type = "HTTP"; Url = "http://localhost:15672" },
    @{ Name = "NATS Monitor"; Host = "localhost"; Port = 8222; Type = "HTTP"; Url = "http://localhost:8222/healthz" },
    @{ Name = "Elasticsearch"; Host = "localhost"; Port = 9200; Type = "HTTP"; Url = "http://localhost:9200" }
)

$passed = 0
$failed = 0

foreach ($s in $services) {
    if ($s.Type -eq "TCP") {
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient
            $async = $tcp.BeginConnect($s.Host, $s.Port, $null, $null)
            $wait = $async.AsyncWaitHandle.WaitOne(3000, $false)
            if ($wait -and $tcp.Connected) {
                $tcp.EndConnect($async)
                $tcp.Close()
                Write-Host "[PASS] $($s.Name) on $($s.Host):$($s.Port) is accepting connections" -ForegroundColor Green
                $passed++
            } else {
                Write-Host "[FAIL] $($s.Name) on $($s.Host):$($s.Port) timed out" -ForegroundColor Red
                $failed++
            }
        } catch {
            Write-Host "[FAIL] $($s.Name) on $($s.Host):$($s.Port) error: $($_.Exception.Message)" -ForegroundColor Red
            $failed++
        }
    } elseif ($s.Type -eq "HTTP") {
        try {
            $resp = Invoke-WebRequest -Uri $s.Url -TimeoutSec 5 -UseBasicParsing
            if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400) {
                Write-Host "[PASS] $($s.Name) on $($s.Url) returned HTTP $($resp.StatusCode)" -ForegroundColor Green
                $passed++
            } else {
                Write-Host "[WARN] $($s.Name) on $($s.Url) returned HTTP $($resp.StatusCode)" -ForegroundColor Yellow
                $passed++
            }
        } catch {
            Write-Host "[FAIL] $($s.Name) on $($s.Url) error: $($_.Exception.Message)" -ForegroundColor Red
            $failed++
        }
    }
}

$summaryColor = "Green"
if ($failed -gt 0) { $summaryColor = "Red" }
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host " Infrastructure Verification Summary: $passed Passed, $failed Failed" -ForegroundColor $summaryColor
Write-Host "============================================================" -ForegroundColor Cyan

if ($failed -gt 0) {
    exit 1
} else {
    exit 0
}
