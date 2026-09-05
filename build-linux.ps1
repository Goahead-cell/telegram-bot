param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",

    [ValidatePattern("^(dev|v[0-9A-Za-z._-]+)$")]
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$projectDir = $PSScriptRoot
$distDir = Join-Path $projectDir "dist"
$bundleName = "telegram-bot-linux-$Arch"
$bundleDir = Join-Path $distDir $bundleName
$archivePath = Join-Path $distDir "$bundleName.tar.gz"
$archiveChecksumPath = "$archivePath.sha256"
$legacyArchivePath = Join-Path $distDir "$bundleName.zip"
$binaryPath = Join-Path $bundleDir "telegram-webhook-bot"
$deployDir = Join-Path $projectDir "deploy"

function Invoke-Go {
    param([string[]]$GoArguments)
    & go @GoArguments
    if ($LASTEXITCODE -ne 0) {
        throw "go command failed with exit code $LASTEXITCODE"
    }
}

Push-Location $projectDir
try {
    Invoke-Go @("test", "./...")
    Invoke-Go @("vet", "./...")

    if (Test-Path -LiteralPath $bundleDir) {
        Remove-Item -LiteralPath $bundleDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $bundleDir -Force | Out-Null

    $oldCGO = [Environment]::GetEnvironmentVariable("CGO_ENABLED", "Process")
    $oldGOOS = [Environment]::GetEnvironmentVariable("GOOS", "Process")
    $oldGOARCH = [Environment]::GetEnvironmentVariable("GOARCH", "Process")
    try {
        $env:CGO_ENABLED = "0"
        $env:GOOS = "linux"
        $env:GOARCH = $Arch
        Invoke-Go @("build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w -X main.version=$Version", "-o", $binaryPath, "./cmd/bot")
    }
    finally {
        if ($null -eq $oldCGO) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $oldCGO }
        if ($null -eq $oldGOOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $oldGOOS }
        if ($null -eq $oldGOARCH) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $oldGOARCH }
    }

    foreach ($fileName in @(
        "install.sh",
        "update.sh",
        "telegram-bot.service",
        "telegram-bot.env.example",
        "Caddyfile.example"
    )) {
        Copy-Item -LiteralPath (Join-Path $deployDir $fileName) -Destination $bundleDir
    }
    Copy-Item -LiteralPath (Join-Path $deployDir "install.sh") -Destination $distDir -Force
    Copy-Item -LiteralPath (Join-Path $deployDir "update.sh") -Destination $distDir -Force

    $hash = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText(
        "$binaryPath.sha256",
        "$hash  telegram-webhook-bot`n",
        [System.Text.Encoding]::ASCII
    )

    if (Test-Path -LiteralPath $archivePath) {
        Remove-Item -LiteralPath $archivePath -Force
    }
    if (Test-Path -LiteralPath $archiveChecksumPath) {
        Remove-Item -LiteralPath $archiveChecksumPath -Force
    }
    if (Test-Path -LiteralPath $legacyArchivePath) {
        Remove-Item -LiteralPath $legacyArchivePath -Force
    }

    if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
        throw "tar is required to create the Linux release archive"
    }
    & tar -C $distDir -czf $archivePath $bundleName
    if ($LASTEXITCODE -ne 0) {
        throw "tar command failed with exit code $LASTEXITCODE"
    }

    $archiveHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText(
        $archiveChecksumPath,
        "$archiveHash  $bundleName.tar.gz`n",
        [System.Text.Encoding]::ASCII
    )

    Write-Host "Build completed:"
    Write-Host "  Version: $Version"
    Write-Host "  Bundle: $bundleDir"
    Write-Host "  Archive: $archivePath"
    Write-Host "  Archive checksum: $archiveChecksumPath"
    Write-Host "  One-click installer: $(Join-Path $distDir 'install.sh')"
    Write-Host "  Binary SHA256: $hash"
}
finally {
    Pop-Location
}
