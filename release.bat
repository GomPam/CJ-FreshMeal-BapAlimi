@echo off
setlocal

echo === CJ-BapAlimi Release ===
echo.

:: 1. gh CLI 설치 확인
where gh >nul 2>&1
if %errorlevel% neq 0 (
    echo [INFO] GitHub CLI가 설치되어 있지 않습니다. 설치합니다...
    winget install --id GitHub.cli --accept-source-agreements --accept-package-agreements
    if %errorlevel% neq 0 (
        echo [ERROR] GitHub CLI 설치 실패. 수동 설치: https://cli.github.com
        exit /b 1
    )
    echo.
    echo [INFO] 설치 완료. 새 터미널에서 다시 실행하세요.
    exit /b 0
)

:: 2. gh 로그인 상태 확인
gh auth status >nul 2>&1
if %errorlevel% neq 0 (
    echo [INFO] GitHub 로그인이 필요합니다.
    gh auth login
    if %errorlevel% neq 0 (
        echo [ERROR] 로그인 실패
        exit /b 1
    )
)

:: 3. 버전 입력
echo.
set /p "VERSION=릴리즈 버전을 입력하세요 (예: 1.0.0): "
if "%VERSION%"=="" (
    echo [ERROR] 버전을 입력해주세요.
    exit /b 1
)

:: 4. 확인
echo.
echo === 릴리즈 정보 ===
echo 버전: v%VERSION%
echo 저장소: GomPam/CJ-FreshMeal-BapAlimi
echo.
set /p "CONFIRM=진행할까요? (Y/N): "
if /i not "%CONFIRM%"=="Y" (
    echo 취소되었습니다.
    exit /b 0
)

:: 5. workflow 실행
echo.
echo [INFO] GitHub Actions 릴리즈 워크플로우 실행 중...
gh workflow run release.yml -f version=%VERSION%
if %errorlevel% neq 0 (
    echo [ERROR] 워크플로우 실행 실패
    exit /b 1
)

echo.
echo [INFO] 워크플로우가 시작되었습니다.
echo [INFO] 진행 상황: https://github.com/GomPam/CJ-FreshMeal-BapAlimi/actions
endlocal
