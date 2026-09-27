@echo off
setlocal
cd /d %~dp0

rem --- pure-Go build: no C++ toolchain needed ---
rem     build.bat          -> tuigo.exe (GUI, single binary)
rem     build.bat test     -> run engine tests
rem     build.bat console  -> tuigo.exe with console (debug)
rem     build.bat res      -> regen app.ico + rsrc.syso (icon/manifest)

if "%1"=="test" (
    go test ./engine/ -v
    exit /b %errorlevel%
)
if "%1"=="console" (
    go build -ldflags "-s -w" -o ..\tuigo.exe .
    exit /b %errorlevel%
)
if "%1"=="res" (
    go run ..\tools\genicon\main.go -o app.ico
    "%GOPATH%\bin\rsrc.exe" -manifest app.manifest -ico app.ico -o rsrc.syso
    if errorlevel 1 rsrc -manifest app.manifest -ico app.ico -o rsrc.syso
    exit /b %errorlevel%
)
go build -ldflags "-H windowsgui -s -w" -o ..\tuigo.exe .
exit /b %errorlevel%
