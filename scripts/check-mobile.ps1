param([switch]$Device, [switch]$Release, [string]$Serial = 'emulator-5554')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$output = Join-Path $repo 'build/mobile-redesign'
New-Item -ItemType Directory -Force $output | Out-Null
& python -P (Join-Path $PSScriptRoot 'check-no-diagnostics.py')
if ($LASTEXITCODE -ne 0) { throw 'Product diagnostics policy failed' }
& python -P (Join-Path $PSScriptRoot 'check-palette.py')
if ($LASTEXITCODE -ne 0) { throw 'Mobile palette differs from Windows' }
Push-Location (Join-Path $repo 'android')
try {
    $tasks = @(':engine:test', ':app:assembleDebug', ':app:lintDebug')
    if ($Device) { $tasks += ':app:assembleDebugAndroidTest' }
    if ($Release) {
        if (!(Test-Path 'keystore.properties')) { throw 'No existing release signing config; do not create or replace a signing key.' }
        $tasks += @(':app:assembleRelease', ':app:lintRelease')
    }
    & .\gradlew.bat @tasks --console=plain 2>&1 | Tee-Object -FilePath (Join-Path $output 'android-final.log')
    if ($LASTEXITCODE -ne 0) { throw 'Android checks failed; see android-final.log' }
    if ($Device) {
        $sdkLine = Get-Content local.properties | Where-Object { $_ -match '^sdk.dir=' } | Select-Object -First 1
        $sdk = ($sdkLine -replace '^sdk.dir=', '') -replace '\\\\', '\'
        $adb = Join-Path $sdk 'platform-tools/adb.exe'
        foreach ($apk in @('app/build/outputs/apk/debug/app-debug.apk', 'app/build/outputs/apk/androidTest/debug/app-debug-androidTest.apk')) {
            & $adb -s $Serial install -r $apk
            if ($LASTEXITCODE -ne 0) { throw "Install failed: $apk" }
        }
        $result = & $adb -s $Serial shell am instrument -w com.tuigo.test/com.tuigo.MobileSmokeTest
        $result | Tee-Object -FilePath (Join-Path $output 'android-device-tests.log')
        if (($result -join "`n") -notmatch 'PASS: 3 exclusive selections') { throw 'Device smoke test failed' }
    }
} finally { Pop-Location }
