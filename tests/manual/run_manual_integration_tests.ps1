$ErrorActionPreference = 'Stop'

$rootDir = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
$outDir = Join-Path $rootDir 'out'

try {
    Push-Location $rootDir

    $goInstalled = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
    if ($goInstalled) {
        Write-Host 'Go detected. Building binaries before running manual integration tests...'

        if (-not (Test-Path -LiteralPath $outDir)) {
            New-Item -ItemType Directory -Path $outDir | Out-Null
        }

        $env:CGO_ENABLED = '0'
        $env:GOOS = 'linux'
        $env:GOARCH = 'amd64'
        & go build -o "$outDir/backup-linux-amd64" .
        & go test -c -tags=integration -o "$outDir/manual-itest-linux-amd64" ./tests/integration

        $env:CGO_ENABLED = '0'
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        & go build -o "$outDir/backup-windows-amd64.exe" .
        & go test -c -tags=integration -o "$outDir/manual-itest-windows-amd64.exe" ./tests/integration
    }

    if (-not $env:BACKUP_BINARY -or [string]::IsNullOrWhiteSpace($env:BACKUP_BINARY)) {
        $env:BACKUP_BINARY = "$rootDir/out/backup-windows-amd64.exe"
    }

    if (-not $env:TEST_BINARY -or [string]::IsNullOrWhiteSpace($env:TEST_BINARY)) {
        $env:TEST_BINARY = "$rootDir/out/manual-itest-windows-amd64.exe"
    }

    if (-not (Test-Path -LiteralPath $env:BACKUP_BINARY)) {
        if (-not $goInstalled) {
            Write-Warning 'Go is not installed and required binaries are missing.'
            Write-Warning 'Build in the dev container first: tests/manual/build_binaries.sh'
        }
        Write-Error "BACKUP_BINARY does not exist: $($env:BACKUP_BINARY)"
    }

    if (-not (Test-Path -LiteralPath $env:TEST_BINARY)) {
        if (-not $goInstalled) {
            Write-Warning 'Go is not installed and required binaries are missing.'
            Write-Warning 'Build in the dev container first: tests/manual/build_binaries.sh'
        }
        Write-Error "TEST_BINARY does not exist: $($env:TEST_BINARY)"
    }

    if (-not $goInstalled) {
        Write-Warning 'Using prebuilt binaries; they may be out of date. Build in the dev container first if needed (tests/manual/build_binaries.sh).'
    }

    if (-not $env:BACKUP_ITEST_PAUSE -or [string]::IsNullOrWhiteSpace($env:BACKUP_ITEST_PAUSE)) {
        $env:BACKUP_ITEST_PAUSE = '1'
    }

    if (-not $env:RESTIC_PASSWORD -or [string]::IsNullOrWhiteSpace($env:RESTIC_PASSWORD)) {
        $env:RESTIC_PASSWORD = 'integration-test-password'
    }

    Write-Host ""
    Write-Host "Running manual integration tests (manifest + restore) with pause setting BACKUP_ITEST_PAUSE=$($env:BACKUP_ITEST_PAUSE)"
    Write-Host "Using backup binary: $($env:BACKUP_BINARY)"
    Write-Host "Using integration test binary: $($env:TEST_BINARY)"
    Write-Host ""

    & $env:TEST_BINARY -test.v -test.run 'TestIntegrationManifestAllCases|TestIntegrationRestoreLatest'
}
finally {
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}
