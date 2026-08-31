param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"
$projectDir = $PSScriptRoot
$distDir = Join-Path $projectDir "dist"
$bundleName = "telegram-bot-linux-$Arch"
$bundleDir = Join-Path $distDir $bundleName
$archivePath = Join-Path $distDir "$bundleName.zip"
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
        Invoke-Go @("build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w", "-o", $binaryPath, "./cmd/bot")
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

    $hash = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText(
        "$binaryPath.sha256",
        "$hash  telegram-webhook-bot`n",
        [System.Text.Encoding]::ASCII
    )

    if (Test-Path -LiteralPath $archivePath) {
        Remove-Item -LiteralPath $archivePath -Force
    }
    Compress-Archive -LiteralPath $bundleDir -DestinationPath $archivePath -CompressionLevel Optimal

    Write-Host "Build completed:"
    Write-Host "  Bundle: $bundleDir"
    Write-Host "  ZIP:    $archivePath"
    Write-Host "  SHA256: $hash"
}
finally {
    Pop-Location
}
