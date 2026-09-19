# 🍽️ skueat (스쿠잇)

성결대학교 주변 맛집 탐색, 실시간 위치 기반 거리 계산, 한 줄 평/별점 평가 및 랜덤 룰렛 추천 서비스입니다.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Gin](https://img.shields.io/badge/Framework-Gin-008ECF?style=flat)
![GORM](https://img.shields.io/badge/ORM-GORM-gray?style=flat)
![SQLite](https://img.shields.io/badge/Database-SQLite-003B57?style=flat&logo=sqlite)
![Fly.io](https://img.shields.io/badge/Deploy-Fly.io-24185B?style=flat&logo=flydotio)

---

## ✨ 주요 기능

- 🗺️ **성결대 중심 인터랙티브 지도**: 카카오 지도 SDK 기반 맛집 마커 표시, 식당 상세 정보 및 도보 거리 제공
- 📍 **GPS 실시간 내 위치 탐색**: 사용자의 현재 위치를 기반으로 주변 맛집을 거리순으로 자동 정렬
- 🎲 **결정 장애 해결 룰렛 추천**: 카페/술집 제외 옵션을 지원하는 부드러운 애니메이션 맛집 추첨기
- 🚙 **원클릭 길찾기 앱 연동**: 카카오맵, 카카오내비, 네이버 지도 앱/웹 크로스플랫폼 딥링크 지원
- 💬 **소셜 로그인 & 한 줄 평/별점**:
  - 카카오 간편 로그인 지원 (OAuth 2.0 및 CSRF 방어)
  - 1인 1식당 1리뷰 정책 (기존 리뷰 작성 시 안전하게 수정 및 평점 자동 재집계)
  - 닉네임 표기 선택권 제공 (마스킹 `홍*동`, 완전 익명 `익명`, 직접 설정 닉네임, 카카오 실명)
- 🌓 **다크 모드 & 서비스 커스텀**: 사용자 디바이스 테마 감지 및 로컬 스토리지 상태 보존
- 📦 **단일 바이너리 패키징 (`go:embed`)**: HTML, CSS/JS 정적 에셋, 초기 맛집 시드 데이터가 바이너리에 내장되어 단독 실행 가능

---

## 🏗️ 아키텍처 구조

Go 표준 프로젝트 레이아웃과 **계층형 아키텍처(Layered Architecture)** 를 채택하여 높은 유지보수성과 테스트 용이성을 제공합니다:

```text
skueat/
├── cmd/
│   ├── main.go            # 서버 진입점 및 DI(의존성 주입), Gin 엔진 구동
│   ├── assets.go          # go:embed 기반 정적 에셋/HTML/시드 데이터 번들링
│   ├── index.html         # 메인 웹앱 템플릿
│   ├── restaurants.json   # 성결대 인근 맛집 초기 데이터
│   └── static/            # CSS, JS(Vanilla), 로고 에셋
├── internal/
│   ├── config/            # 환경 변수 로드 및 설정 관리
│   ├── model/             # GORM 엔티티 및 DTO 정의
│   ├── repository/        # SQLite/GORM 데이터 접근 계층 및 원자적 평점 재계산
│   ├── service/           # 비즈니스 로직 (리뷰 Upsert, OAuth 연동, 마스킹)
│   ├── handler/           # HTTP 라우팅, 세션 검증, 응답 제어
│   └── utils/             # 이름 마스킹 등 유틸리티
├── Dockerfile             # 경량화 멀티스테이지 도커 빌드 설정
├── fly.toml               # Fly.io 클라우드 배포 및 영구 볼륨 설정
└── .github/workflows/     # CI 테스트 및 CD 배포 자동화
```

---

## 🚀 빠른 시작 (Getting Started)

### 1. 사전 요구 사항
- [Go 1.24+](https://golang.org/dl/)
- [카카오 디벨로퍼스](https://developers.kakao.com) 애플리케이션 등록
  - JavaScript 키 (지도 렌더링용)
  - REST API 키 (카카오 로그인용)
  - Redirect URI 등록: `http://localhost:8080/auth/kakao/callback`

### 2. 환경 설정
`.env.example` 파일을 복사하여 `.env` 파일을 생성하고 키를 입력합니다:

```bash
cp .env.example .env
```

### 3. 로컬 서버 실행
Go 1.16+ `embed`가 적용되어 있어, 프로젝트 루트나 cmd 어느 위치에서든 즉시 실행할 수 있습니다:

```bash
go run ./cmd
```

브라우저에서 `http://localhost:8080`에 접속합니다.

---

## 🧪 테스트 실행

검색은 브라우저에서 공백·영문 대소문자 무시, 여러 단어 AND 검색, 초성 검색을 함께 적용합니다.
이름 일치도를 우선하고 같은 순위에서는 거리순으로 표시합니다. 입력 후 180ms에 갱신하며 한글 조합 중에는 기다립니다.
식당 목록은 페이지를 열 때 가져오며, 외부 변경 사항은 새로고침으로 반영합니다.
초성 변환에는 로컬에 포함한 es-hangul 2.4.0을 사용합니다 (`cmd/static/vendor`, MIT 라이선스 포함).
검색 테스트: `node --test tests/search.test.mjs`

### 기존 리뷰와 로그인 전환
- 계정 ID가 없는 이전 세션은 재로그인이 필요합니다.
- 기존 닉네임 기반 리뷰는 내용과 평점 집계를 유지하며, 소유권을 확인할 수 없어 새 계정에 자동 연결하지 않습니다.
- 새로 작성한 리뷰부터 인증된 계정 ID로 수정 권한을 확인합니다. 기존 리뷰와 별개로 첫 리뷰가 등록될 수 있습니다.
- DB의 `account_id`는 시작 시 자동 추가됩니다. 기존 `user_id`를 복사해 채우지 마세요. 닉네임 충돌로 다른 사용자의 리뷰가 연결될 수 있습니다.

```bash
go test -v ./...
```

---

## 🚢 배포 (Docker & Fly.io)

### Docker 로컬 빌드 및 실행
```bash
docker build -t skueat .
docker run -p 8080:8080 -e KAKAO_API_KEY=xxx -e REST_API_KEY=yyy skueat
```

### Fly.io 배포
본 프로젝트는 SQLite 영구 데이터 유지를 위해 Fly.io 볼륨 마운트(`/app/data`)가 사전 구성되어 있습니다:
```bash
flyctl deploy
```
GitHub 저장소에 Push 시 GitHub Actions 워크플로우를 통해 테스트 검증 후 자동 배포됩니다.
