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

wails build -clean
if %errorlevel% neq 0 (
    echo [ERROR] 빌드 실패
    exit /b 1
)

echo.
echo === 빌드 완료 ===
echo 출력: build\bin\CJ-BapAlimi.exe
endlocal
