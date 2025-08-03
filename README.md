# Trace 모듈

Go 언어로 작성된 고성능 Trace 로그 수집 및 비동기 저장 모듈입니다.

## 📋 목차

- [개요](#개요)
- [주요 기능](#주요-기능)
- [설치 및 설정](#설치-및-설정)
- [사용법](#사용법)
- [API 문서](#api-문서)
- [성능 최적화](#성능-최적화)
- [문제 해결](#문제-해결)

## 🎯 개요

Trace 모듈은 웹 애플리케이션의 HTTP 요청을 추적하고 분석하기 위한 고성능 로깅 시스템입니다. 비동기 배치 처리와 메모리 최적화를 통해 대용량 트래픽에서도 안정적으로 작동합니다.

### 핵심 특징

- ⚡ **고성능**: 비동기 배치 처리로 성능 최적화
- 🔧 **유연성**: 다양한 사용자 정보 추출 방식 지원
- 💾 **안정성**: 재시도 로직과 메모리 관리
- 📊 **분석**: 상세한 요청 정보 수집

## 🚀 주요 기능

### 1. HTTP 요청 추적

- 요청 경로, 메서드, 상태 코드 기록
- 응답 시간 측정 및 분석
- 클라이언트 IP 및 User-Agent 정보 수집

### 2. 사용자 행동 분석

- 사용자별 API 호출 패턴 추적
- 시간별 사용자 행동 분석
- 세션 기반 사용자 흐름 추적

### 3. 성능 모니터링

- API별 응답 시간 분석
- 병목 지점 식별
- 에러율 및 성능 지표 수집

### 4. 커스터마이징 가능한 미들웨어

- 다양한 사용자 정보 추출 방식
- 조건부 필터링
- 커스텀 Trace ID 생성

## 📦 설치 및 설정

### 1. 로컬 설치

#### 의존성 설치

```bash
go mod tidy
```

### 2. Docker 설치

#### Docker 이미지 빌드

```bash
docker build -t trace-app .
```

#### Docker Compose로 실행

```bash
# 기본 실행 (SQLite 사용)
docker-compose up -d

# 개발 모드 실행
docker-compose --profile dev up -d

# PostgreSQL과 함께 실행
docker-compose --profile postgres up -d

# 모든 서비스 실행 (PostgreSQL + Redis + Nginx)
docker-compose --profile postgres --profile redis --profile nginx up -d
```

#### Docker Compose 서비스 관리

```bash
# 서비스 상태 확인
docker-compose ps

# 로그 확인
docker-compose logs -f trace-app

# 서비스 중지
docker-compose down

# 볼륨과 함께 완전 삭제
docker-compose down -v
```

### 3. Kubernetes 설치

#### 사전 요구사항

```bash
# kubectl 설치 확인
kubectl version --client

# 클러스터 연결 확인
kubectl cluster-info
```

#### 기본 배포

```bash
# 배포 스크립트 실행
cd k8s
./deploy.sh

# 수동 배포
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f persistent-volume.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
```

#### Ingress와 모니터링 포함 배포

```bash
# Ingress 포함
./deploy.sh --with-ingress

# 모니터링 포함
./deploy.sh --with-monitoring

# 모든 기능 포함
./deploy.sh --with-ingress --with-monitoring
```

#### Kustomize 사용

```bash
# 기본 배포
kubectl apply -k k8s/

# 환경별 배포
kubectl apply -k k8s/overlays/production/
kubectl apply -k k8s/overlays/development/
```

### 4. 데이터베이스 설정

```go
import (
"gorm.io/driver/sqlite" // SQLite 사용 예시
"gorm.io/gorm"
)

// 데이터베이스 연결
db, err := gorm.Open(sqlite.Open("trace.db"), &gorm.Config{})
if err != nil {
log.Fatal("Failed to connect to database:", err)
}
```

### 3. Trace 모듈 초기화

```go
import "trace/internal/trace"

// 설정 구성
cfg := trace.Config{
DB:              db,
FlushInterval:   5 * time.Second,
BatchSize:       100,
BufferSize:      1000,
MaxOpenConns:    10,
MaxIdleConns:    5,
ConnMaxLifetime: time.Hour,
}

// 모듈 시작
if err := trace.Start(cfg); err != nil {
log.Fatal("Failed to start trace module:", err)
}
```

### 4. Docker 환경 설정

#### 환경 변수 설정

```bash
# .env 파일 생성
cat > .env << EOF
GIN_MODE=release
DB_TYPE=sqlite
DB_PATH=/app/data/trace.db
FLUSH_INTERVAL=5s
BATCH_SIZE=100
BUFFER_SIZE=1000
EOF
```

#### 데이터 디렉토리 생성

```bash
# 데이터 및 로그 디렉토리 생성
mkdir -p data logs
```

## 🛠️ 사용법

### 1. 기본 미들웨어 사용

```go
import (
"github.com/gin-gonic/gin"
"trace/internal/trace"
)

func main() {
r := gin.Default()

// 기본 미들웨어 추가 (쿼리 파라미터 방식)
r.Use(trace.Middleware())

r.Run(":8080")
}
```

### 2. 커스터마이징된 미들웨어 사용

#### 헤더 기반 사용자 정보 추출

```go
headerMiddleware := trace.MiddlewareWithConfig(
trace.WithUserIDExtractor(func(c *gin.Context) string {
return c.GetHeader("X-User-ID")
}),
trace.WithTokenExtractor(func (c *gin.Context) string {
return c.GetHeader("X-Access-Token")
}),
)

r.Use(headerMiddleware)
```

#### JWT 토큰 기반 사용자 정보 추출

```go
jwtMiddleware := trace.MiddlewareWithConfig(
trace.WithUserIDExtractor(func(c *gin.Context) string {
auth := c.GetHeader("Authorization")
if strings.HasPrefix(auth, "Bearer ") {
token := strings.TrimPrefix(auth, "Bearer ")
// JWT 토큰에서 사용자 ID 추출 로직
return extractUserIDFromJWT(token)
}
return ""
}),
trace.WithTokenExtractor(func (c *gin.Context) string {
auth := c.GetHeader("Authorization")
if strings.HasPrefix(auth, "Bearer ") {
return strings.TrimPrefix(auth, "Bearer ")
}
return ""
}),
)
```

#### 조건부 필터링

```go
filteredMiddleware := trace.MiddlewareWithConfig(
trace.WithFilter(func(c *gin.Context) bool {
// 특정 경로만 추적
importantPaths := []string{"/api/users", "/api/orders"}
for _, path := range importantPaths {
if strings.HasPrefix(c.Request.URL.Path, path) {
return true
}
}
return false
}),
)
```

#### 커스텀 Trace ID 생성

```go
customMiddleware := trace.MiddlewareWithConfig(
trace.WithTraceIDGenerator(func(userID, token string) string {
return fmt.Sprintf("trace-%s-%d", userID, time.Now().UnixNano())
}),
)
```

## 📊 API 문서

### 데이터베이스 스키마

Trace 모듈은 자동으로 다음 테이블을 생성합니다:

```sql
CREATE TABLE trace_steps
(
    trace_id    VARCHAR(255) INDEX, -- Trace ID (인덱스)
    user_id     VARCHAR(255) INDEX, -- 사용자 ID (인덱스)
    path        VARCHAR(255),       -- API 경로
    method      VARCHAR(10),        -- HTTP 메서드
    status_code INTEGER,            -- HTTP 상태 코드
    latency_ms  BIGINT,             -- 응답 시간 (밀리초)
    ip          VARCHAR(45),        -- 클라이언트 IP
    user_agent  TEXT,               -- 사용자 에이전트
    created_at  BIGINT INDEX        -- 타임스탬프 (인덱스)
);
```

### 설정 옵션

| 옵션                | 설명            | 기본값  | 권장값      |
|-------------------|---------------|------|----------|
| `FlushInterval`   | 로그 플러시 간격     | 5초   | 3-10초    |
| `BatchSize`       | 배치 처리 크기      | 100  | 50-200   |
| `BufferSize`      | 메모리 버퍼 크기     | 1000 | 500-2000 |
| `MaxOpenConns`    | 최대 DB 연결 수    | 10   | 5-20     |
| `MaxIdleConns`    | 최대 유휴 DB 연결 수 | 5    | 3-10     |
| `ConnMaxLifetime` | DB 연결 수명      | 1시간  | 30분-2시간  |

## ⚡ 성능 최적화

### 1. 메모리 관리

- **슬라이스 재사용**: `buf = buf[:0]`로 메모리 재할당 방지
- **배치 크기 제한**: 500개로 제한하여 메모리 사용량 관리
- **버퍼 크기 조정**: 시스템 메모리에 맞게 조정

### 2. 데이터베이스 최적화

- **인덱스 활용**: 자주 조회하는 필드에 인덱스 설정
- **배치 처리**: 대량 데이터를 효율적으로 저장
- **연결 풀 관리**: 적절한 연결 수로 성능 최적화

### 3. 비동기 처리

- **논블로킹 채널**: `select` 문으로 버퍼 오버플로우 방지
- **고루틴 활용**: 메인 스레드 블로킹 방지
- **재시도 로직**: 일시적 오류에 대한 복원력

## 🔧 문제 해결

### 1. 일반적인 문제들

#### 로그가 저장되지 않는 경우

```bash
# 데이터베이스 연결 확인
sqlite3 trace.db ".tables"

# 로그 확인
sqlite3 trace.db "SELECT COUNT(*) FROM trace_steps;"
```

#### 성능 문제

```go
// 버퍼 크기 증가
cfg := trace.Config{
BufferSize: 2000, // 기본값 1000에서 증가
BatchSize:  200, // 기본값 100에서 증가
}
```

#### 메모리 사용량 문제

```go
// 배치 크기 감소
cfg := trace.Config{
BatchSize: 50, // 기본값 100에서 감소
BufferSize: 500, // 기본값 1000에서 감소
}
```

### 2. 모니터링

#### 로그 확인

```bash
# 실시간 로그 모니터링
tail -f your-app.log | grep "trace"

# 성공적인 플러시 확인
grep "successfully flushed" your-app.log

# Docker 컨테이너 로그 확인
docker-compose logs -f trace-app
```

### 3. Docker 관련 문제 해결

#### 컨테이너가 시작되지 않는 경우

```bash
# 컨테이너 상태 확인
docker-compose ps

# 상세 로그 확인
docker-compose logs trace-app

# 컨테이너 재시작
docker-compose restart trace-app
```

#### 볼륨 권한 문제

```bash
# 호스트 디렉토리 권한 설정
sudo chown -R $USER:$USER data/ logs/

# Docker 볼륨 권한 확인
docker-compose exec trace-app ls -la /app/data
```

#### 포트 충돌 문제

```bash
# 사용 중인 포트 확인
netstat -tulpn | grep :8080

# 다른 포트로 변경
# docker-compose.yml에서 ports 섹션 수정
# - "8081:8080"  # 8080 대신 8081 사용
```

### 4. Kubernetes 관련 문제 해결

#### Pod가 시작되지 않는 경우

```bash
# Pod 상태 확인
kubectl get pods -n trace-system

# Pod 상세 정보 확인
kubectl describe pod -n trace-system -l app=trace

# Pod 로그 확인
kubectl logs -n trace-system -l app=trace

# Pod 재시작
kubectl rollout restart deployment/trace-app -n trace-system
```

#### PersistentVolume 문제

```bash
# PV/PVC 상태 확인
kubectl get pv,pvc -n trace-system

# PV 상세 정보 확인
kubectl describe pv trace-data-pv

# PVC 상세 정보 확인
kubectl describe pvc trace-data-pvc -n trace-system
```

#### Service 연결 문제

```bash
# Service 상태 확인
kubectl get svc -n trace-system

# Service 상세 정보 확인
kubectl describe svc trace-service -n trace-system

# 포트 포워딩으로 테스트
kubectl port-forward -n trace-system svc/trace-service 8080:80
```

#### Ingress 문제

```bash
# Ingress 상태 확인
kubectl get ingress -n trace-system

# Ingress 상세 정보 확인
kubectl describe ingress trace-ingress -n trace-system

# Ingress 컨트롤러 로그 확인
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

#### HPA 문제

```bash
# HPA 상태 확인
kubectl get hpa -n trace-system

# HPA 상세 정보 확인
kubectl describe hpa trace-hpa -n trace-system

# 메트릭 서버 확인
kubectl get pods -n kube-system | grep metrics-server
```

#### 데이터베이스 상태 확인

```sql
-- 최근 로그 확인
SELECT *
FROM trace_steps
ORDER BY created_at DESC LIMIT 10;

-- 사용자별 호출 수 확인
SELECT user_id, COUNT(*) as calls
FROM trace_steps
GROUP BY user_id
ORDER BY calls DESC;

-- API별 응답 시간 확인
SELECT path, method, AVG(latency_ms) as avg_latency
FROM trace_steps
GROUP BY path, method
ORDER BY avg_latency DESC;
```

## 📝 예제

### 완전한 예제 애플리케이션

```go
package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"trace/internal/trace"
)

func main() {
	// 데이터베이스 연결
	db, err := gorm.Open(sqlite.Open("trace.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Trace 모듈 초기화
	cfg := trace.Config{
		DB:              db,
		FlushInterval:   5 * time.Second,
		BatchSize:       100,
		BufferSize:      1000,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	if err := trace.Start(cfg); err != nil {
		log.Fatal("Failed to start trace module:", err)
	}

	// Gin 라우터 설정
	r := gin.Default()

	// 기본 미들웨어 추가
	r.Use(trace.Middleware())

	// API 엔드포인트
	r.GET("/api/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Users API"})
	})

	r.GET("/api/orders", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Orders API"})
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
```

### 테스트 방법

```bash
# 기본 테스트
curl "http://localhost:8080/api/users?user_id=123&access_token=abc123"

# 헤더 기반 테스트
curl -H "X-User-ID: 123" -H "X-Access-Token: abc123" \
     "http://localhost:8080/api/orders"

# JWT 기반 테스트
curl -H "Authorization: Bearer user123.token.signature" \
     "http://localhost:8080/api/users"
```

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.
