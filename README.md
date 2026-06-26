# Image API (Go + Gin)

시놀로지 NAS 환경을 기준으로 한 이미지 업로드/서빙 API 서비스입니다.

- 업로드/삭제 API: Go(Gin) 앱 처리
- 이미지 조회(GET): Nginx 직접 서빙
- 배포: Docker Compose

## 주요 기능

- `POST /api/images/upload` 이미지 업로드
- `DELETE /api/images/:subdir/:filename` 이미지 삭제
- `GET /api/images/:subdir/:filename` 이미지 조회 (Nginx 직접 서빙)
- `GET /api/images/health` 헬스체크
- `X-API-Key` 인증(선택)
- MIME 기반 이미지 타입 검증
- 경로 순회 공격 방지

## 프로젝트 구조

```text
.
├── main.go
├── go.mod
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── .env.example
└── internal
    ├── config
    │   └── config.go
    ├── handler
    │   └── image.go
    ├── middleware
    │   └── auth.go
    └── util
        └── util.go
```

## 환경변수

`.env.example`를 복사해 `.env`를 생성하세요.

| 변수명 | 기본값 | 설명 |
|---|---|---|
| `UPLOAD_DIR` | `/data/images` | 이미지 저장 경로 |
| `MAX_FILE_SIZE` | `20971520` | 업로드 최대 크기(byte), 기본 20MB |
| `CORS_ORIGINS` | `*` | 허용 Origin(쉼표 구분) |
| `BASE_URL` | `http://localhost:8080` | 업로드 응답에 포함할 base URL |
| `PORT` | `8080` | 앱 포트 |
| `API_KEY` | `` | 업로드/삭제 인증 키 (비어있으면 인증 비활성화) |

## 로컬 실행 (Go)

> Go 1.22 이상 권장 (`go.mod` 기준)

```bash
cp .env.example .env
mkdir -p ./tmp/images
UPLOAD_DIR=./tmp/images PORT=8080 go run main.go
```

## Docker 실행

시놀로지 기준 이미지 저장 볼륨은 `/volume1/images`를 사용합니다.

```bash
cp .env.example .env
docker compose up -d --build
```

- App: `http://localhost:8080`
- Nginx: `http://localhost:8888`

## API 명세 요약

### 1) 이미지 업로드

`POST /api/images/upload`

- Header: `X-API-Key: <your_key>` (`API_KEY` 설정 시 필수)
- Body: `multipart/form-data`, 필드명 `file`

예시:

```bash
curl -X POST http://localhost:8080/api/images/upload \
  -H "X-API-Key: your_key" \
  -F "file=@photo.jpg"
```

성공 응답 예시:

```json
{
  "filename": "abc123def456789012345678901234ab.jpg",
  "path": "ab/abc123def456789012345678901234ab.jpg",
  "url": "http://localhost:8080/api/images/ab/abc123def456789012345678901234ab.jpg",
  "size": 204800,
  "mime_type": "image/jpeg",
  "md5": "d41d8cd98f00b204e9800998ecf8427e"
}
```

### 2) 이미지 조회

`GET /api/images/:subdir/:filename`

예시:

```bash
curl http://localhost:8080/api/images/ab/abc123.jpg
```

### 3) 이미지 삭제

`DELETE /api/images/:subdir/:filename`

예시:

```bash
curl -X DELETE http://localhost:8080/api/images/ab/abc123.jpg \
  -H "X-API-Key: your_key"
```

### 4) 헬스체크

`GET /api/images/health`

예시:

```bash
curl http://localhost:8080/api/images/health
```

## Nginx 라우팅 역할

- `GET /api/images/{subdir}/{filename}`: Nginx가 `/data/images`에서 직접 서빙
- `POST /api/images/upload`: Go 앱으로 프록시
- `DELETE /api/images/{subdir}/{filename}`: Go 앱으로 프록시
- `GET /api/images/health`: Go 앱으로 프록시

## 참고

- 업로드 파일명은 UUID 기반 32자리 hex + 확장자로 생성됩니다.
- 저장 경로는 파일명 앞 2글자를 서브디렉토리로 사용합니다.
- 예: `ab/abc123def456789012345678901234ab.jpg`
