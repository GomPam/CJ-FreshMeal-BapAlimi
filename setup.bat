@echo off
setlocal enabledelayedexpansion

echo === CJ-BapAlimi 개발 환경 설정 ===
echo.

:: ── Go 확인 / 자동 설치 ──
set "GO_SDK_DIR=%USERPROFILE%\go-sdk"
set "GO_ROOT=%GO_SDK_DIR%\go"

where go >nul 2>&1
if %errorlevel% neq 0 (
    if exist "%GO_ROOT%\bin\go.exe" (
        echo [OK] Go found at %GO_ROOT%\bin\go.exe
    ) else (
        echo [INFO] Go 미설치 - 자동 다운로드합니다...
        echo.

        :: 최신 버전 조회
        for /f "delims=" %%V in ('powershell -NoProfile -Command "(Invoke-WebRequest -Uri 'https://go.dev/VERSION?m=text' -UseBasicParsing).Content.Split([char]10)[0]"') do set "GO_VER=%%V"
        if "!GO_VER!"=="" (
            echo [ERROR] Go 버전 조회 실패
            exit /b 1
        )
        echo [INFO] 최신 버전: !GO_VER!

        set "GO_ZIP=!GO_VER!.windows-amd64.zip"
        set "GO_URL=https://go.dev/dl/!GO_ZIP!"
        set "GO_DL=%TEMP%\!GO_ZIP!"

        echo [INFO] 다운로드: !GO_URL!
        powershell -NoProfile -Command "Invoke-WebRequest -Uri '!GO_URL!' -OutFile '!GO_DL!' -UseBasicParsing"
        if not exist "!GO_DL!" (
            echo [ERROR] 다운로드 실패
            exit /b 1
        )
        echo [OK] 다운로드 완료

        if not exist "%GO_SDK_DIR%" mkdir "%GO_SDK_DIR%"
        echo [INFO] 압축 해제 중... (1-2분 소요)
        powershell -NoProfile -Command "Expand-Archive -Path '!GO_DL!' -DestinationPath '%GO_SDK_DIR%' -Force"
        del "!GO_DL!" >nul 2>&1

        if not exist "%GO_ROOT%\bin\go.exe" (
            echo [ERROR] Go 설치 실패
            exit /b 1
        )
        echo [OK] Go 설치 완료: %GO_ROOT%
    )
    set "PATH=%GO_ROOT%\bin;%USERPROFILE%\go\bin;%PATH%"
) else (
    echo [OK] Go found
)
go version

:: ── Node.js 확인 ──
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARN] Node.js 미설치 - https://nodejs.org/ 에서 설치하세요
    echo        (Wails 빌드에 필요합니다)
    exit /b 1
) else (
    echo [OK] Node.js found
)
node --version

:: ── Wails CLI 설치 ──
set "PATH=%USERPROFILE%\go\bin;%PATH%"
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

:: ── Go 의존성 설치 ──
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
