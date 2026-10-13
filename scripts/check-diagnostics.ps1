$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
python -P (Join-Path $PSScriptRoot 'check-no-diagnostics.py')
if ($LASTEXITCODE -ne 0) { throw 'Product diagnostics policy failed' }
Push-Location (Join-Path $repo 'app')
try {
    $previousGoWork = $env:GOWORK
    $env:GOWORK = 'off'
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Windows/engine/updater tests failed' }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Windows/engine/updater vet failed' }
} finally {
    $env:GOWORK = $previousGoWork
    Pop-Location
}
