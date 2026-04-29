@echo off
setlocal

echo === CJ-BapAlimi 개발 환경 설정 ===
echo.

:: Go 확인
where go >nul 2>&1
if %errorlevel% neq 0 (
    if exist "%USERPROFILE%\go-sdk\go\bin\go.exe" (
        echo [OK] Go found at %USERPROFILE%\go-sdk\go\bin\go.exe
        set "PATH=%USERPROFILE%\go-sdk\go\bin;%PATH%"
    ) else (
        echo [ERROR] Go 미설치 - https://go.dev/dl/ 에서 설치하세요
        exit /b 1
    )
) else (
    echo [OK] Go found
)
go version

:: Node.js 확인
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARN] Node.js 미설치 - https://nodejs.org/ 에서 설치하세요
    echo        (Wails 빌드에 필요합니다)
    exit /b 1
) else (
    echo [OK] Node.js found
)
node --version

:: Wails CLI 설치
where wails >nul 2>&1
if %errorlevel% neq 0 (
    echo [INFO] Wails CLI 설치 중...
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if %errorlevel% neq 0 (
        echo [ERROR] Wails CLI 설치 실패
        exit /b 1
    )
    echo [OK] Wails CLI 설치 완료
) else (
    echo [OK] Wails CLI found
)

:: Go 의존성 설치
echo.
echo [INFO] Go 의존성 설치 중...
go mod tidy
if %errorlevel% neq 0 (
    echo [ERROR] 의존성 설치 실패
    exit /b 1
)

echo.
echo === 설치 완료 ===
echo 빌드: build.bat
echo 개발: wails dev
endlocal
