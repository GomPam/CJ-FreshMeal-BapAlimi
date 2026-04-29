@echo off
setlocal

:: PATH 설정 (사용자 설치 Go 지원)
if exist "%USERPROFILE%\go-sdk\go\bin\go.exe" (
    set "PATH=%USERPROFILE%\go-sdk\go\bin;%USERPROFILE%\go\bin;%PATH%"
)

echo === CJ-BapAlimi 빌드 ===
echo.

where wails >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Wails CLI가 없습니다. setup.bat을 먼저 실행하세요.
    exit /b 1
)

:: 버전 추출
for /f "delims=" %%V in ('powershell -NoProfile -Command "(Get-Content wails.json | ConvertFrom-Json).info.productVersion"') do set "APP_VERSION=%%V"
if "%APP_VERSION%"=="" (
    echo [WARN] 버전 추출 실패, dev로 빌드합니다.
    set "APP_VERSION=dev"
)
echo 버전: %APP_VERSION%
echo.

wails build -clean -ldflags "-X main.appVersion=%APP_VERSION%"
if %errorlevel% neq 0 (
    echo [ERROR] 빌드 실패
    exit /b 1
)

echo.
echo === 빌드 완료 (v%APP_VERSION%) ===
echo 출력: build\bin\CJ-BapAlimi.exe
endlocal
