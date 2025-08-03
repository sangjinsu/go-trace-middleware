// trace 패키지 - Trace 로그 수집 및 비동기 저장 모듈
package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Step 로그 구조체
type Step struct {
	TraceID    string `gorm:"index"` // 인덱스 추가로 검색 성능 향상
	UserID     string `gorm:"index"` // 유저별 검색을 위한 인덱스
	Path       string // API 경로
	Method     string // HTTP 메서드 (GET, POST, PUT, DELETE 등)
	StatusCode int    // HTTP 상태 코드
	LatencyMs  int64  // 응답 시간 (밀리초)
	IP         string // 클라이언트 IP
	UserAgent  string // 사용자 에이전트
	CreatedAt  int64  `gorm:"index"` // 타임스탬프 (Unix timestamp)
}

// Config 설정 구조체
type Config struct {
	DB              *gorm.DB
	FlushInterval   time.Duration
	BatchSize       int
	BufferSize      int
	MaxOpenConn     int
	MaxIdleConn     int
	ConnMaxLifetime time.Duration
}

// MiddlewareConfig 미들웨어 설정 구조체
type MiddlewareConfig struct {
	// 사용자 ID 추출 함수
	UserIDExtractor func(c *gin.Context) string
	// 토큰 추출 함수
	TokenExtractor func(c *gin.Context) string
	// Trace ID 생성 함수
	TraceIDGenerator func(userID, token string) string
	// 필터링 함수 (true면 로그 수집, false면 스킵)
	Filter func(c *gin.Context) bool
}

// 기본 추출 함수들
func defaultUserIDExtractor(c *gin.Context) string {
	return c.Query("user_id")
}

func defaultTokenExtractor(c *gin.Context) string {
	return c.Query("access_token")
}

func defaultTraceIDGenerator(userID, token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s:%s", userID, hex.EncodeToString(h[:]))
}

func defaultFilter(c *gin.Context) bool {
	return true
}

// MiddlewareOption 함수형 옵션 타입
type MiddlewareOption func(*MiddlewareConfig)

// WithUserIDExtractor 사용자 ID 추출 함수 설정
func WithUserIDExtractor(extractor func(c *gin.Context) string) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.UserIDExtractor = extractor
	}
}

// WithTokenExtractor 토큰 추출 함수 설정
func WithTokenExtractor(extractor func(c *gin.Context) string) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.TokenExtractor = extractor
	}
}

// WithTraceIDGenerator Trace ID 생성 함수 설정
func WithTraceIDGenerator(generator func(userID, token string) string) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.TraceIDGenerator = generator
	}
}

// WithFilter 필터링 함수 설정
func WithFilter(filter func(c *gin.Context) bool) MiddlewareOption {
	return func(config *MiddlewareConfig) {
		config.Filter = filter
	}
}

var buffer chan Step

// Start 워커 초기화
func Start(cfg Config) error {
	buffer = make(chan Step, cfg.BufferSize)

	sqlDB, err := cfg.DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := cfg.DB.AutoMigrate(&Step{}); err != nil {
		return err
	}

	go startWorker(cfg.DB, cfg.FlushInterval, cfg.BatchSize)
	return nil
}

// Middleware - 기본 Gin 미들웨어 (기존 호환성 유지)
func Middleware() gin.HandlerFunc {
	return MiddlewareWithConfig()
}

// MiddlewareWithConfig - 설정 가능한 Gin 미들웨어
func MiddlewareWithConfig(options ...MiddlewareOption) gin.HandlerFunc {
	config := &MiddlewareConfig{
		UserIDExtractor:  defaultUserIDExtractor,
		TokenExtractor:   defaultTokenExtractor,
		TraceIDGenerator: defaultTraceIDGenerator,
		Filter:           defaultFilter,
	}

	// 옵션 적용
	for _, option := range options {
		option(config)
	}

	return func(c *gin.Context) {
		// 필터링 체크
		if !config.Filter(c) {
			c.Next()
			return
		}

		userID := config.UserIDExtractor(c)
		token := config.TokenExtractor(c)

		if userID == "" || token == "" {
			c.Next()
			return
		}

		traceID := config.TraceIDGenerator(userID, token)

		start := time.Now()
		c.Next()
		latency := time.Since(start).Milliseconds()

		step := Step{
			TraceID:    traceID,
			UserID:     userID,
			Path:       c.FullPath(),
			Method:     c.Request.Method,
			StatusCode: c.Writer.Status(),
			LatencyMs:  latency,
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			CreatedAt:  time.Now().Unix(),
		}

		select {
		case buffer <- step:
			// 정상 저장
		default:
			// 버퍼가 가득 찬 경우 드롭
		}
	}
}

func startWorker(db *gorm.DB, interval time.Duration, batchSize int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	buf := make([]Step, 0, batchSize*2) // 초기 용량 설정

	for {
		select {
		case log, ok := <-buffer:
			if !ok {
				// 채널이 닫힌 경우
				if len(buf) > 0 {
					flush(db, buf)
				}
				return
			}

			buf = append(buf, log)
			if len(buf) >= batchSize {
				flush(db, buf)
				buf = buf[:0] // 슬라이스 재사용
			}
		case <-ticker.C:
			if len(buf) > 0 {
				flush(db, buf)
				buf = buf[:0] // 슬라이스 재사용
			}
		}
	}
}

func flush(db *gorm.DB, logs []Step) {
	if len(logs) == 0 {
		return
	}

	go func(logs []Step) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic during trace flush: %v", r)
			}
		}()

		// 재시도 로직 (최대 3회)
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := flushWithRetry(db, logs, attempt); err != nil {
				if attempt == maxRetries {
					log.Printf("failed to flush after %d attempts: %v", maxRetries, err)
					return
				}
				// 재시도 전 잠시 대기
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}

			log.Printf("successfully flushed %d trace logs", len(logs))
			return
		}
	}(logs)
}

// flushWithRetry - 재시도 로직이 포함된 flush 함수
func flushWithRetry(db *gorm.DB, logs []Step, attempt int) error {
	// 배치 크기 설정 (메모리 효율성을 위해 500으로 제한)
	batchSize := min(500, len(logs))

	// 배치 단위로 저장
	for i := 0; i < len(logs); i += batchSize {
		end := min(i+batchSize, len(logs))

		batch := logs[i:end]
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			return fmt.Errorf("failed to create batch %d-%d (attempt %d): %v", i, end-1, attempt, err)
		}
	}

	return nil
}
