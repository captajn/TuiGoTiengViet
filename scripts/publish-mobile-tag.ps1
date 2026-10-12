param([Parameter(Mandatory)][string]$Tag, [Parameter(Mandatory)][string]$Commit, [switch]$CheckOnly)
$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    & python -P scripts/mobile-release.py metadata --tag $Tag
    if ($LASTEXITCODE -ne 0) { throw 'Invalid mobile tag/version' }
    $target = & git rev-parse --verify "$Commit^{commit}"
    if ($LASTEXITCODE -ne 0) { throw 'Target commit does not exist' }
    $local = & git rev-parse --verify --quiet "refs/tags/$Tag"
    $hasLocal = $LASTEXITCODE -eq 0
    if ($hasLocal -and $local -ne $target) { throw 'Local tag already points elsewhere; do not push or overwrite it.' }
    $remote = & git ls-remote --exit-code origin "refs/tags/$Tag"
    $remoteStatus = $LASTEXITCODE
    if ($remoteStatus -notin @(0, 2)) { throw 'Cannot inspect remote tag' }
    if ($remoteStatus -eq 0 -and ($remote -split '\s+')[0] -ne $target) { throw 'Remote tag already points elsewhere; explicit user decision required.' }
    if ($CheckOnly) { Write-Output "PASS mobile tag state: $Tag -> $target"; return }
    if (!$hasLocal) {
        & git tag $Tag $target
        if ($LASTEXITCODE -ne 0) { throw 'Tag creation failed; push cancelled' }
    }
    & git push origin "refs/tags/$Tag"
    if ($LASTEXITCODE -ne 0) { throw 'Tag push failed' }
} finally { Pop-Location }
