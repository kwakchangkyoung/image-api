# Image Go API (Go + Gin) — 개발 명세서

## 프로젝트 개요

시놀로지 NAS에서 운영하는 이미지 업로드/서빙 서버.
외부 공개 서비스용이며 DSM 역방향 프록시 + Let's Encrypt 인증서를 사용한다.
GET 이미지 서빙은 Nginx가 직접 처리하고, Go 앱은 업로드/삭제 API만 담당한다.
Docker 최종 이미지는 `scratch` 기반으로 ~10MB 수준을 유지한다.

---

## 기술 스택

- **언어**: Go 1.22
- **프레임워크**: Gin v1.10.0
- **컨테이너**: Docker (멀티스테이지 빌드) + Docker Compose
- **리버스 프록시**: Nginx (Alpine)
- **주요 라이브러리**:
  - `github.com/gin-gonic/gin` — HTTP 프레임워크
  - `github.com/google/uuid` — UUID 파일명 생성
  - `github.com/joho/godotenv` — .env 파일 로드

---

## 디렉토리 구조

```
image_go_api/
├── main.go                        # 진입점, Gin 라우터 설정
├── go.mod
├── go.sum
├── internal/
│   ├── config/
│   │   └── config.go              # 환경변수 기반 설정
│   ├── handler/
│   │   └── image.go               # Upload / Get / Delete / Health 핸들러
│   ├── middleware/
│   │   └── auth.go                # API Key 인증 미들웨어
│   └── util/
│       └── util.go                # MIME 감지, 파일명 생성, MD5, 경로 검증
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
└── .env.example
```

---

## 코딩 컨벤션

- **중괄호 스타일**: BSD 스타일 (여는 중괄호를 다음 줄에)
- **들여쓰기**: 탭 (Go 표준)
- **포맷터**: `gofmt` 적용
- **패키지 구조**: `internal/` 하위로 외부 노출 차단

BSD 스타일 예시:
```go
func Foo() string
{
    return "bar"
}

if err != nil
{
    return err
}
```

---

## 환경변수 (.env)

| 변수명 | 기본값 | 설명 |
|--------|--------|------|
| `UPLOAD_DIR` | `/data/images` | 이미지 저장 경로 (NAS 볼륨 마운트 경로) |
| `MAX_FILE_SIZE` | `20971520` | 최대 업로드 크기 (bytes, 기본 20MB) |
| `CORS_ORIGINS` | `*` | 허용 CORS 출처 (쉼표 구분, 복수 가능) |
| `BASE_URL` | `http://localhost:8080` | 업로드 응답에 포함할 이미지 base URL |
| `PORT` | `8080` | 서버 리슨 포트 |
| `API_KEY` | `""` | 업로드/삭제 인증 키 (빈 값이면 인증 비활성화) |

---

## Config 구조체 (`internal/config/config.go`)

```go
type Config struct {
    UploadDir   string
    MaxFileSize int64
    AllowedMIME map[string]string  // mime type → 확장자
    CORSOrigins []string
    BaseURL     string
    Port        string
    APIKey      string
}
```

`AllowedMIME` 초기값:
```go
map[string]string{
    "image/jpeg": "jpg",
    "image/png":  "png",
    "image/gif":  "gif",
    "image/webp": "webp",
}
```

환경변수 로드: `godotenv.Load()` → `os.Getenv()` 순서로 처리.
`API_KEY` 미설정(빈 문자열)이면 인증 미들웨어가 스킵한다.

---

## API 명세

### POST `/api/images/upload`

이미지 파일 업로드.

- **인증**: `X-API-Key` 헤더 필수 (API_KEY 설정 시)
- **Content-Type**: `multipart/form-data`
- **파라미터**: form field명 `file`
- **허용 MIME**: `image/jpeg`, `image/png`, `image/gif`, `image/webp`
- **파일 크기 제한**: `MAX_FILE_SIZE` (기본 20MB)

**처리 순서**:
1. `c.Request.FormFile("file")` 로 파일 수신
2. `header.Size` 로 크기 검사 → 초과 시 413
3. `io.ReadAll()` 로 전체 읽기
4. `http.DetectContentType(data)` 로 실제 MIME 감지 → 불허 시 415
5. UUID hex 파일명 생성, 앞 2글자로 서브디렉토리 결정
6. `os.MkdirAll()` → `os.WriteFile()` 로 저장
7. 응답 JSON 반환

**응답 (200)**
```json
{
  "filename": "abc123def456789012345678901234ab.jpg",
  "path": "ab/abc123def456789012345678901234ab.jpg",
  "url": "https://images.example.com/api/images/ab/abc123def456789012345678901234ab.jpg",
  "size": 204800,
  "mime_type": "image/jpeg",
  "md5": "d41d8cd98f00b204e9800998ecf8427e"
}
```

**에러 응답**
| 코드 | 사유 |
|------|------|
| 400 | form file 없음 |
| 401 | API Key 없거나 불일치 |
| 413 | 파일 크기 초과 |
| 415 | 지원하지 않는 이미지 형식 |
| 500 | 디렉토리 생성 또는 파일 저장 실패 |

---

### GET `/api/images/{subdir}/{filename}`

이미지 파일 조회. **Nginx가 직접 서빙** (Go 앱 미경유).
Go 핸들러는 Nginx 우회 시 fallback 역할만 한다.

- **인증**: 없음 (공개)
- **캐시 헤더**: `Cache-Control: public, max-age=2592000, immutable`
- **구현**: `c.File(filePath)` 사용

---

### DELETE `/api/images/{subdir}/{filename}`

이미지 파일 삭제.

- **인증**: `X-API-Key` 헤더 필수
- **구현**: `os.Remove(filePath)`
- **응답 (200)**
```json
{ "message": "삭제되었습니다.", "filename": "abc123.jpg" }
```

---

### GET `/api/images/health`

헬스체크. 인증 없음.

```json
{ "status": "ok", "upload_dir": "/data/images" }
```

---

## 주요 구현 사항

### 1. 파일명 생성 및 저장 구조 (`internal/util/util.go`)

```
UUID → strings.ReplaceAll(uuid.New().String(), "-", "")
     → 32자리 hex: abc123def456789012345678901234ab

파일명: abc123def456789012345678901234ab.jpg
서브디렉토리: filename[:2] → "ab"

저장 경로: /data/images/ab/abc123def456789012345678901234ab.jpg
URL 경로:  /api/images/ab/abc123def456789012345678901234ab.jpg
```

### 2. MIME 타입 감지

`http.DetectContentType(data)` 사용. 파일 헤더(magic bytes) 기반으로 실제 타입 감지.
확장자 위조 업로드 방지.

### 3. 경로 순회 공격 방지 (`SafePath`)

```go
func SafePath(base, subdir, filename string) (string, bool)
{
    if strings.Contains(subdir, "..") || strings.Contains(filename, "..") {
        return "", false
    }
    path := filepath.Join(base, subdir, filename)
    if !strings.HasPrefix(path, filepath.Clean(base)+"/") {
        return "", false
    }
    return path, true
}
```

### 4. API Key 인증 미들웨어 (`internal/middleware/auth.go`)

- 헤더명: `X-API-Key`
- `APIKey` 가 빈 문자열이면 `c.Next()` 즉시 호출 (개발 환경 스킵)
- 불일치 시 `c.AbortWithStatusJSON(401, ...)` 으로 이후 핸들러 차단
- 적용: `POST /upload`, `DELETE /{subdir}/{filename}`
- 미적용: `GET /{subdir}/{filename}`, `GET /health`

### 5. CORS 미들웨어 (`main.go` 인라인)

- `CORSOrigins` 에 `*` 포함 시 모든 출처 허용
- 아니면 요청 Origin 과 일치 여부 확인
- `OPTIONS` 메서드 → 204 즉시 응답

### 6. Nginx 역할 분담

```
GET /api/images/{subdir}/{filename}    →  Nginx 직접 서빙 (sendfile, alias /data/images/)
POST /api/images/upload                →  proxy_pass http://app:8080
DELETE /api/images/{subdir}/{filename} →  proxy_pass http://app:8080
GET /api/images/health                 →  proxy_pass http://app:8080
```

---

## Docker 구성

### Dockerfile (멀티스테이지 빌드)

**빌드 스테이지**: `golang:1.22-alpine`
```
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o image_api .
```

**실행 스테이지**: `scratch` (OS 없는 최소 이미지, ~10MB)
```
COPY --from=builder /build/image_api /image_api
EXPOSE 8080
ENTRYPOINT ["/image_api"]
```

### docker-compose.yml

서비스 2개:

**app**
- build: 현재 디렉토리
- 포트: `8080:8080`
- 볼륨: `/volume1/images:/data/images` (rw)
- env_file: `.env`

**nginx**
- 이미지: `nginx:alpine`
- 포트: `8888:80`
- 볼륨:
  - `/volume1/images:/data/images` (ro)
  - `./nginx.conf:/etc/nginx/conf.d/default.conf` (ro)
- depends_on: app

### nginx.conf 라우팅 규칙

- `POST /api/images/upload` → `proxy_pass http://app:8080`, `client_max_body_size 20m`
- `GET /api/images/health` → `proxy_pass http://app:8080`
- `DELETE` 요청 → `proxy_pass http://app:8080`
- `GET /api/images/{subdir}/{filename}` → `alias /data/images/`, `expires 30d`, `try_files $uri =404`

---

## 라우터 구성 (`main.go`)

```go
api := r.Group("/api/images")
{
    api.POST("/upload", auth, img.Upload)
    api.GET("/health", img.Health)
    api.GET("/:subdir/:filename", img.Get)
    api.DELETE("/:subdir/:filename", auth, img.Delete)
}
```

`auth` = `middleware.APIKeyAuth(cfg.APIKey)` 반환 `gin.HandlerFunc`

---

## 시놀로지 배포 절차

1. NAS에 `/volume1/images` 디렉토리 생성
2. `.env.example` → `.env` 복사 후 값 수정
3. `docker-compose up -d` 실행
4. DSM 제어판 → 로그인 포털 → 역방향 프록시 설정:
   - 소스: `HTTPS`, `images.yourdomain.com`, `443`
   - 대상: `HTTP`, `localhost`, `8888`
5. DSM 제어판 → 보안 → 인증서: Let's Encrypt 발급 후 역방향 프록시에 할당

---

## 로컬 개발

```bash
cd ~/workspace/image_go_api
go mod tidy
mkdir -p ./tmp/images
UPLOAD_DIR=./tmp/images go run main.go
```

API 테스트:
```bash
# 업로드
curl -X POST http://localhost:8080/api/images/upload \
  -H "X-API-Key: your_key" \
  -F "file=@photo.jpg"

# 조회
curl http://localhost:8080/api/images/ab/abc123.jpg

# 삭제
curl -X DELETE http://localhost:8080/api/images/ab/abc123.jpg \
  -H "X-API-Key: your_key"
```

---

## go.mod 의존성

```
module image_go_api

go 1.22

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
)
```
