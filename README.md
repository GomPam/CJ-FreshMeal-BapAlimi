# CJ-BapAlimi (밥알리미)

CJ 프레시밀 식단을 시스템 트레이에서 조회하고, Microsoft Teams 채팅으로 자동 전송하는 Windows 데스크톱 앱.

## 주요 기능

- **식단 조회** — CJ 프레시밀 API에서 오늘/주간 식단을 가져와 카드 UI로 표시 (조식/중식/석식)
- **Teams 자동 전송** — 설정된 시간에 Adaptive Card 형태로 Teams 채팅에 식단 발송
- **시스템 트레이 상주** — 트레이 아이콘 클릭으로 팝업, 백그라운드 동작
- **다크/라이트 테마** — 시스템 설정 연동 또는 수동 전환

## 기술 스택

| 구성 | 기술 |
|---|---|
| 백엔드 | Go |
| 프레임워크 | Wails v2 (Go + WebView2) |
| 프론트엔드 | HTML/CSS/JS (프레임워크 없음) |
| 인증 | MSAL Go (Device Code Flow) |
| 빌드 결과 | 단일 exe (~15MB) |

## 프로젝트 구조

```
CJ-BapAlimi/
├── main.go          # 진입점, Wails 초기화
├── app.go           # 핵심 로직 (식단 API, Teams 인증/전송, 스케줄러)
├── tray.go          # 시스템 트레이, 윈도우 관리
├── icon.png         # 앱/트레이 아이콘 원본
├── wails.json       # Wails 프로젝트 설정
├── go.mod / go.sum  # Go 의존성
├── setup.bat        # 개발 환경 설정 (최초 1회)
├── build.bat        # 빌드 스크립트
├── build/
│   ├── appicon.png          # 앱 아이콘
│   └── windows/
│       ├── icon.ico         # Windows exe 아이콘
│       ├── info.json        # 버전 정보
│       └── wails.exe.manifest
└── frontend/
    ├── index.html   # UI 구조
    ├── style.css    # 스타일 (다크/라이트 테마)
    └── main.js      # 프론트엔드 로직
```

## 개발 환경 설정

### 요구 사항

- Go 1.24+
- Node.js 18+
- Wails CLI v2

### 최초 설정

```bat
setup.bat
```

Go, Node.js 설치 여부를 확인하고 Wails CLI 및 Go 의존성을 설치한다.

### 빌드

```bat
build.bat
```

빌드 결과: `build\bin\CJ-BapAlimi.exe`

### 개발 모드

```bat
wails dev
```

핫 리로드 지원. 프론트엔드/백엔드 변경 시 자동 반영.

## 사용 방법

1. `CJ-BapAlimi.exe` 실행 → 시스템 트레이에 아이콘 표시 (중복 실행 시 자동 종료)
2. 트레이 아이콘 클릭 → 오늘 식단 확인
3. 설정(⚙) → Teams 로그인 (Device Code Flow)
4. 전송 대상(채팅) 선택 + 전송 시간 설정
5. 설정된 시간에 자동으로 Teams 채팅에 식단 전송
6. 같은 날 재전송 시 기존 메시지를 삭제하고 새로 발송 (알림 재수신)

## 창 동작

- 트레이 클릭 시 주 모니터 우하단에 고정 표시
- 창 드래그/리사이즈 불가 (트레이 팝업 방식)
- 중복 실행 방지 (Windows named mutex)

## 키보드 단축키

| 키 | 동작 |
|---|---|
| `ESC` | 메인 뷰: 창 숨김 / 설정 뷰: 메인으로 복귀 |

## 식단 API

| 엔드포인트 | 설명 |
|---|---|
| `GET /meal/v1/today-all-meal?storeIdx={idx}` | 오늘 식단 |
| `GET /meal/v1/week-meal?storeIdx={idx}&weekType=1` | 이번주 |
| `GET /meal/v1/week-meal?storeIdx={idx}&weekType=2` | 다음주 |
| `GET /store/v1/search-store?page={n}&schKey={keyword}&isList=false` | 식당 검색 |

base URL: `https://front.cjfreshmeal.co.kr`

## 썸네일 자동 갱신

메인 화면에서 오늘 날짜를 보고 있을 때, 아래 시간대에 식단 데이터를 자동 새로고침한다:

- **중식**: 11:15 ~ 11:35 (5분 간격)
- **석식**: 17:15 ~ 17:35 (5분 간격)

## 기본 전송 시간

최초 설치 또는 설정 초기화 시 기본 전송 시간: **11:20**, **17:20**

## 설정 초기화

설정 화면 헤더의 초기화 버튼(↺)을 누르면 모든 설정을 기본값으로 되돌린다:

- Teams 인증 (토큰 삭제)
- 전송 대상 / 전송 시간
- 전송 기록 / 로그
- 시작 프로그램 등록 해제

## 자동 실행

설정 화면에서 "시작 프로그램 등록" 토글을 켜면 Windows 로그인 시 자동 실행된다.

- 레지스트리 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`에 exe 경로 등록
- exe 위치가 변경된 경우 앱 시작 시 레지스트리 경로를 자동 갱신
- 토글 OFF 시 레지스트리 항목 제거

## Teams 연동

- Azure AD 앱 등록 없이 Graph PowerShell 공용 Client ID 사용
- 권한: `Chat.ReadWrite` (delegated)
- 인증 흐름: Device Code Flow → Silent Token Refresh
- 토큰 캐시: `%LOCALAPPDATA%/CJ-BapAlimi/token_cache.json`

## 로컬 데이터

`%LOCALAPPDATA%/CJ-BapAlimi/` 에 저장:

| 파일 | 용도 |
|---|---|
| `config.json` | 전송 대상, 시간 설정 |
| `token_cache.json` | MSAL 토큰 캐시 |
| `sent_records.json` | 당일 전송 기록 (중복 방지) |
| `app.log` | 앱 로그 |

## 면책 조항

- 이 프로젝트는 **CJ 프레시밀과 무관한 개인 프로젝트**이며, CJ 프레시밀의 공식 승인이나 제휴를 받지 않았다.
- 식단 데이터는 CJ 프레시밀의 공개 API를 통해 조회하며, API의 가용성이나 정확성을 보장하지 않는다.
- 앱 내 표시되는 CJ 프레시밀 로고 및 상표는 해당 권리자의 자산이다.
- Microsoft Teams 연동은 Microsoft Graph API의 공용 Client ID를 사용하며, Microsoft의 공식 승인을 받지 않았다.
- 본 소프트웨어에 포함된 오픈소스 라이브러리는 각각의 라이선스(MIT, Apache 2.0, BSD)를 따른다.
- 본 소프트웨어는 있는 그대로(AS-IS) 제공되며, 사용으로 인한 어떠한 책임도 지지 않는다.
