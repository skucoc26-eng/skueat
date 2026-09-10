<div align="center">

# 🍽️ skueat (스쿠잇)

**성결대학교(SKU) 주변 맛집 지도 & 큐레이션 서비스**  
"오늘 뭐 먹지?" 고민을 한 방에 해결해 주는 학생들을 위한 스마트 맛집 지도 앱

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Gin Framework](https://img.shields.io/badge/Gin-v1.10-008ECF?style=flat&logo=go)](https://gin-gonic.com/)
[![SQLite](https://img.shields.io/badge/SQLite-GORM-003B57?style=flat&logo=sqlite)](https://gorm.io/)
[![Kakao Maps](https://img.shields.io/badge/Kakao_Maps-API-FFCD00?style=flat&logo=kakao)](https://apis.map.kakao.com/)
[![Fly.io](https://img.shields.io/badge/Deploy-Fly.io-24185B?style=flat&logo=flydotio)](https://fly.io/)

</div>

---

## 📌 서비스 소개

**skueat**은 성결대학교 주변의 음식점, 카페, 술집 정보를 한눈에 확인하고, 간편하게 검색 및 추천을 받을 수 있는 웹 애플리케이션입니다.  
카카오맵 기반의 직관적인 지도와 모바일 친화적인 바텀시트 UI, 그리고 재미있는 룰렛 추천과 카카오 소셜 로그인 기반의 리뷰/별점 시스템을 제공합니다.

---

## ✨ 주요 기능

### 🗺️ 인터랙티브 맛집 지도 & 리스트
- **Kakao Maps API 연동**: 성결대 인근 맛집 위치를 지도 마커로 표시하고, 마커 클릭 시 식당 상세 정보 확인 가능
- **모바일 최적화 바텀시트 UI**: 드래그 제스처 및 반응형 리스트 레이아웃으로 모바일 환경에서도 쾌적하게 사용
- **카테고리 필터링**: 전체, 한식, 중식, 일식, 양식, 치킨, 분식, 고기, 국수, 카페, 술집 등 원터치 필터링
- **실시간 키워드 검색**: 식당명, 주소, 메뉴 키워드로 원하는 장소 즉시 검색

### 📍 GPS 내 위치 찾기 & 길찾기 연동
- **내 위치 탐색**: 브라우저 Geolocation API를 통해 지도상에서 현재 내 위치를 파악하고 중심 이동
- **원클릭 길찾기**: 설정에서 선호하는 내비게이션(카카오맵, 카카오내비, 네이버 지도)을 지정하여 길찾기 바로 연결

### 🎲 '오늘 뭐 먹지?' 룰렛 랜덤 추천
- 메뉴 결정이 어려운 순간, 슬롯머신 룰렛 애니메이션을 통한 즉석 추천 모달 지원
- **스마트 제외 옵션**: '☕ 카페 제외', '🍺 술집 제외' 체크박스로 식사 메뉴만 쏙 골라 추천

### 💬 카카오 로그인 & 생생한 한 줄 평(리뷰)
- **카카오 OAuth 소셜 로그인**: 카카오 계정으로 간편 로그인 (7일간 유지되는 안전한 쿠키 세션)
- **별점 평가 (1~5점) 및 한 줄 평 작성**: 실제 방문자들의 생생한 리뷰 공유
- **개인정보 보호 닉네임 옵션**:
  - `마스킹` (예: 김*수, 남**수)
  - `익명`
  - `실명`
  - `커스텀 닉네임` (최대 10자)

### ⚙️ 사용자 맞춤 설정 & 테마
- **다크 모드 / 라이트 모드 (🌓)**: 테마 토글 버튼 지원 및 사용자 브라우저(LocalStorage) 설정 저장
- **지도 마커 토글**: 필요에 따라 지도 마커를 켜고 끌 수 있는 기능 제공

---

## 🛠️ 기술 스택 (Tech Stack)

| 구분 | 기술 | 설명 |
| :--- | :--- | :--- |
| **Backend** | **Go 1.24+**, **Gin Web Framework** | 고성능 경량 REST API 서버 구축 |
| **Session** | **gin-contrib/sessions** (Cookie Store) | 카카오 로그인 세션 관리 (유효기간 7일) |
| **Database** | **SQLite3**, **GORM** (`glebarez/sqlite`) | CGO가 필요 없는 순수 Go SQLite 드라이버 및 ORM |
| **Frontend** | **HTML5**, **CSS3**, **Vanilla JavaScript (ES6+)** | 외부 프레임워크 없는 순수 모던 웹 프론트엔드 |
| **Map & Auth** | **Kakao Maps API**, **Kakao OAuth 2.0** | 지도 렌더링, 위치 마커, 카카오 간편 로그인 |
| **Deployment** | **Docker** (Multi-stage Alpine), **Fly.io** | 가벼운 컨테이너 빌드 및 영구 볼륨(`skueat_data`) 배포 |
| **CI / CD** | **GitHub Actions** | `main`/`master` 브랜치 푸시 시 Fly.io 자동 배포 |

---

## 📁 프로젝트 구조 (Project Structure)

```text
skueat/
├── .github/
│   └── workflows/
│       └── fly-deploy.yml    # GitHub Actions를 통한 Fly.io 자동 배포 워크플로우
├── cmd/
│   ├── static/
│   │   ├── app.js            # 지도 초기화, 이벤트 제어, 룰렛, 리뷰 렌더링 스크립트
│   │   ├── style.css         # 반응형 레이아웃, 바텀시트, 다크모드 스타일시트
│   │   └── logo.svg          # skueat 서비스 로고
│   ├── .env                  # 로컬 환경 변수 설정 파일 (API 키 등)
│   ├── db.go                 # SQLite DB 연결, AutoMigrate, 초기 데이터 로더 (JSON)
│   ├── index.html            # 메인 SPA 템플릿 및 모달 (룰렛, 설정) 구조
│   ├── main.go               # Gin 서버 초기화, 라우터, 카카오 OAuth, API 엔드포인트
│   ├── mask_test.go          # 닉네임 마스킹 로직 단위 테스트
│   ├── restaurants.db        # SQLite 데이터베이스 파일 (로컬 실행 시 생성)
│   └── restaurants.json      # 성결대 주변 초기 맛집 데이터셋
├── Dockerfile                # 멀티스테이지 경량 도커 이미지 빌드 정의
├── fly.toml                  # Fly.io 클라우드 배포 및 볼륨 마운트 설정
├── go.mod                    # Go 모듈 의존성 정의
├── go.sum                    # Go 모듈 체크섬
└── README.md                 # 프로젝트 문서
```

---

## 🔌 주요 API 명세 (API Endpoints)

| Method | Endpoint | Description | 인증 여부 |
| :--- | :--- | :--- | :---: |
| `GET` | `/` | 메인 서비스 웹 페이지 렌더링 | X |
| `GET` | `/api/restaurants` | 맛집 목록 조회 (`?category=한식&search=키워드`) | X |
| `GET` | `/api/restaurants/random` | 무작위 맛집 1곳 추천 | X |
| `GET` | `/api/reviews` | 특정 식당의 리뷰 및 별점 목록 조회 (`?restaurant_id={id}`) | X |
| `POST` | `/api/rate` | 식당 별점 및 한 줄 평 등록 (`score`, `comment`, `author_type` 등) | **필수 (로그인)** |
| `GET` | `/login/kakao` | 카카오 OAuth 인증 시작 (카카오 로그인 페이지로 리다이렉트) | X |
| `GET` | `/auth/kakao/callback` | 카카오 OAuth 인가 코드 수신 및 세션 생성 | X |
| `GET` | `/logout` | 세션 파기 및 로그아웃 | X |

---

## 🚀 시작하기 (Getting Started)

### 1. 사전 준비 (Prerequisites)
- [Go](https://go.dev/dl/) 1.24 이상 설치
- [Kakao Developers](https://developers.kakao.com/) 계정 및 애플리케이션 등록
  - **JavaScript 키** 발급 (지도 렌더링용)
  - **REST API 키** 발급 (카카오 소셜 로그인용)
  - 카카오 로그인 활성화 및 Redirect URI 등록: `{APP_DOMAIN}/auth/kakao/callback`
  - Web 플랫폼 사이트 도메인 등록: `{APP_DOMAIN}`

### 2. 환경 변수 설정 (`cmd/.env`)
`cmd` 디렉터리 내에 `.env` 파일을 생성하고 아래 항목들을 입력합니다.

```env
# 카카오 API 키
KAKAO_API_KEY=your_kakao_javascript_key
REST_API_KEY=your_kakao_rest_api_key

# 서비스 도메인 및 서버 설정
APP_DOMAIN=http://localhost:8080
APP_HOST=0.0.0.0
APP_PORT=8080

# 세션 및 데이터베이스 (선택 사항)
SESSION_SECRET=your_secret_key_here
DATABASE_PATH=restaurants.db
```

### 3. 의존성 설치 및 로컬 실행

```bash
# 1. 의존성 모듈 다운로드
go mod tidy

# 2. cmd 디렉터리로 이동 후 서버 실행
cd cmd
go run main.go db.go

# 또는 프로젝트 루트에서 빌드 후 실행
go run ./cmd
```

서버가 구동되면 브라우저에서 `http://localhost:8080`으로 접속합니다.

---

## 🐳 Docker 및 클라우드 배포

### Docker 빌드 및 로컬 실행

```bash
# 도커 이미지 빌드
docker build -t skueat .

# 컨테이너 실행 (환경변수 전달)
docker run -d -p 8080:8080 \
  -e KAKAO_API_KEY="your_javascript_key" \
  -e REST_API_KEY="your_rest_api_key" \
  -e APP_DOMAIN="http://localhost:8080" \
  --name skueat-app skueat
```

### Fly.io 배포

이 프로젝트는 `fly.toml`과 GitHub Actions 워크플로우를 통해 Fly.io 배포가 구성되어 있습니다.
- SQLite 데이터 보존을 위해 Fly.io 볼륨(`skueat_data` -> `/app/data`)을 마운트하여 사용합니다.
- `main` 또는 `master` 브랜치에 코드를 푸시하면 GitHub Actions에 의해 자동으로 배포가 진행됩니다.

```bash
# Fly CLI를 통한 수동 배포
fly deploy
```

---

## 🧪 테스트 실행

```bash
# 전체 테스트 실행
go test ./...

# cmd 패키지 상세 테스트
go test -v ./cmd
```
