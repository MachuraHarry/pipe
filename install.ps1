# Pipe (SPR) installer for Windows (PowerShell 5.1+)
#
# Downloads the latest (or pinned) Pipe release for Windows, verifies its
# SHA256 checksum and installs pipe.exe into your user PATH.
#
# Usage (Run in PowerShell):
#   irm https://pipe-lang.com/install.ps1 | iex
#
# Parameters:
#   -Version <tag>   Release tag to install, e.g. v0.9.3 (default: latest)
#   -Prefix <path>   Install directory (default: $env:LOCALAPPDATA\Programs\pipe\bin)
#   -SkipPath        Do not modify the user PATH (used for CI smoke tests)

[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$Prefix = "",
    [switch]$SkipPath
)

$ErrorActionPreference = "Stop"

$Repo = "MachuraHarry/pipe"
$BinName = "pipe.exe"
$Artifact = "pipe-windows-amd64.tar.gz"
$Base = "https://github.com/$Repo/releases"

if (-not $Prefix) {
    $Prefix = Join-Path $env:LOCALAPPDATA "Programs\pipe\bin"
}

if ($Version -eq "latest") {
    $Url = "$Base/latest/download/$Artifact"
} else {
    $Url = "$Base/download/$Version/$Artifact"
}

function Die {
    param([string]$Message)
    Write-Host "error: $Message" -ForegroundColor Red
    exit 1
}

Write-Host "Downloading Pipe $Version (windows/amd64)"
$tmp = Join-Path $env:TEMP ("pipe-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    $tarball = Join-Path $tmp $Artifact
    Invoke-WebRequest -Uri $Url -OutFile $tarball -UseBasicParsing

    $shaFile = Join-Path $tmp "$Artifact.sha256"
    try {
        Invoke-WebRequest -Uri "$Url.sha256" -OutFile $shaFile -UseBasicParsing
        $expected = ((Get-Content $shaFile).Split(" ")[0]).Trim().ToLowerInvariant()
        $actual = (Get-FileHash -Path $tarball -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($expected -ne $actual) {
            Die "SHA256 verification failed for $Artifact"
        }
        Write-Host "SHA256 verified"
    } catch {
        Write-Host "warning: no SHA256 checksum available, skipping verification"
    }

    if (Get-Command tar.exe -ErrorAction SilentlyContinue) {
        tar -xzf $tarball -C $tmp | Out-Null
    } else {
        Die "tar.exe not found (Windows 10 1803+ required); extract $tarball manually and place $BinName in your PATH"
    }

    $binary = Join-Path $tmp $BinName
    if (-not (Test-Path $binary)) {
        Die "archive did not contain a $BinName binary"
    }

    New-Item -ItemType Directory -Path $Prefix -Force | Out-Null
    Copy-Item -Path $binary -Destination (Join-Path $Prefix $BinName) -Force

    $inPath = $false
    if (-not $SkipPath) {
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -and ($userPath -split ";" | Where-Object { $_ -and $_.TrimEnd("\") -eq $Prefix.TrimEnd("\") })) {
            $inPath = $true
        }
        if (-not $inPath) {
            $newPath = if ($userPath) { "$userPath;$Prefix" } else { $Prefix }
            [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
            Write-Host "Added $Prefix to your user PATH"
        }
    }

    Write-Host ""
    Write-Host "Pipe installed: $(Join-Path $Prefix $BinName)"
    Write-Host ""
    if (-not $inPath) {
        Write-Host "NOTE: Open a NEW PowerShell window, or refresh PATH in this one:"
        Write-Host '  $env:Path = [Environment]::GetEnvironmentVariable("Path", "User") + ";" + $env:Path'
        Write-Host ""
    }
    $full = Join-Path $Prefix $BinName
    $help = & $full -h 2>&1 | Select-Object -First 1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Verified: $help"
    } else {
        Write-Host $help
    }
} finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
