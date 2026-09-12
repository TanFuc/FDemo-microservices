[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateSet("ON", "OFF", "STATUS")]
    [string]$Action
)

$ErrorActionPreference = "Stop"

$Distro = "Ubuntu-22.04"
$RepoWindows = "C:\Users\nguye\OneDrive\Desktop\Project\FDemo-microservices"
$RepoWsl = "/mnt/c/Users/nguye/OneDrive/Desktop/Project/FDemo-microservices"
$ComposeFile = "deploy/compose/docker-compose.databases.yml"
$ProjectName = "fdemo-microservices"

function Invoke-Compose {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    & wsl.exe -d $Distro --cd $RepoWsl --exec docker compose `
        -p $ProjectName -f $ComposeFile @Arguments

    if ($LASTEXITCODE -ne 0) {
        throw "Docker Compose failed with exit code $LASTEXITCODE."
    }
}

function Get-RunningCount {
    $ids = & wsl.exe -d $Distro --cd $RepoWsl --exec docker compose `
        -p $ProjectName -f $ComposeFile ps -q --status running

    if ($LASTEXITCODE -ne 0) {
        return 0
    }

    return @($ids | Where-Object { $_ -and $_.Trim() }).Count
}

switch ($Action) {
    "ON" {
        $running = Get-RunningCount
        if ($running -gt 0) {
            Write-Host "FDemo is already running ($running containers)." -ForegroundColor Yellow
            exit 0
        }

        if (-not (Get-Command wt.exe -ErrorAction SilentlyContinue)) {
            throw "Windows Terminal (wt.exe) was not found."
        }

        $infraCommand = "wsl.exe -d $Distro --cd $RepoWsl --exec docker compose -p $ProjectName -f $ComposeFile up"
        $shellCommand = "Set-Location -LiteralPath '$RepoWindows'; Write-Host 'FDemo project shell ready.' -ForegroundColor Green"

        Start-Process wt.exe -ArgumentList @(
            "-w", "FDemo",
            "new-tab",
            "--title", "FDemo Infrastructure",
            "powershell.exe", "-NoExit", "-Command", $infraCommand
        )

        Start-Sleep -Milliseconds 800

        Start-Process wt.exe -ArgumentList @(
            "-w", "FDemo",
            "new-tab",
            "--title", "FDemo Shell",
            "powershell.exe", "-NoExit", "-Command", $shellCommand
        )

        Write-Host "FDemo startup requested in one Windows Terminal window with two tabs." -ForegroundColor Green
    }

    "OFF" {
        $running = Get-RunningCount
        if ($running -eq 0) {
            Write-Host "FDemo is already stopped." -ForegroundColor Green
            exit 0
        }

        Invoke-Compose stop
        Write-Host "FDemo stopped. Volumes and data were preserved." -ForegroundColor Green
    }

    "STATUS" {
        $running = Get-RunningCount
        Write-Host "FDemo running containers: $running" -ForegroundColor Cyan

        if ($running -gt 0) {
            Invoke-Compose ps
        }

        $memory = & wsl.exe -d $Distro --exec sh -lc "free -m | sed -n '2p'"
        if ($LASTEXITCODE -eq 0 -and $memory) {
            Write-Host "WSL memory (MB): $memory"
        }

        $allContainers = & wsl.exe -d $Distro --exec docker ps --format "{{.Names}}|{{.Status}}"
        if ($LASTEXITCODE -eq 0) {
            $otherContainers = @($allContainers | Where-Object {
                $_ -and $_ -notmatch "^tafu-"
            })

            if ($otherContainers.Count -gt 0) {
                Write-Host "Other running containers (not controlled by this script):" -ForegroundColor Yellow
                $otherContainers | ForEach-Object { Write-Host "  $_" }
            }
        }
    }
}
