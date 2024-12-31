@echo off
setlocal enabledelayedexpansion

REM Application name
set APP_NAME=cacheserver

REM Version and build info
for /f "tokens=*" %%i in ('git describe --tags --always') do set VERSION=%%i
for /f "tokens=*" %%i in ('powershell -command "Get-Date -Format \"yyyy-MM-ddTHH:mm\""') do set BUILD=%%i

REM Output directory
set OUTPUT_DIR=binwin

REM Platforms to build for
set PLATFORMS=windows/amd64 linux/386 linux/amd64 linux/arm/7 linux/arm64 darwin/amd64

REM Build flags
set LDFLAGS=-s -w -X main.GitCommit=%VERSION% -X main.BuildTime=%BUILD%

REM Clean up the output directory
echo Cleaning up...
if exist %OUTPUT_DIR% rmdir /s /q %OUTPUT_DIR%
mkdir %OUTPUT_DIR%
echo %OUTPUT_DIR%

set PLATFORMS=windows/amd64 linux/386 linux/amd64 linux/arm/7 linux/arm64 darwin/amd64
REM Build for each platform
for %%p in (%PLATFORMS%) do (
    echo %%p
    for /f "tokens=1-3 delims=/" %%a in ("%%p") do (
        set OS="%%a"
        set ARCH="%%b"
        set ARM="%%c"
        echo "OS=" %%a "ARCH=" %%b "ARM=" %%c "aaaa"
        echo "OS=" %OS% "ARCH=" %ARCH% "ARM=" %ARM% "bbbb"
    )
)