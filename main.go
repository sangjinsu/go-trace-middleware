package main

import (
	"fmt"
	"log"
	"strings"
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
		MaxOpenConn:     10,
		MaxIdleConn:     5,
		ConnMaxLifetime: time.Hour,
	}

	if err = trace.Start(cfg); err != nil {
		log.Fatal("Failed to start trace module:", err)
	}

	// Gin 라우터 설정
	r := gin.Default()

	// 예제 1: 기본 미들웨어 (기존 호환성)
	r.GET("/basic", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Basic middleware",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 예제 2: 헤더에서 사용자 ID 추출
	headerMiddleware := trace.MiddlewareWithConfig(
		trace.WithUserIDExtractor(func(c *gin.Context) string {
			return c.GetHeader("X-User-ID")
		}),
		trace.WithTokenExtractor(func(c *gin.Context) string {
			return c.GetHeader("X-Access-Token")
		}),
	)

	r.GET("/header", headerMiddleware, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Header-based middleware",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 예제 3: JWT 토큰에서 사용자 ID 추출
	jwtMiddleware := trace.MiddlewareWithConfig(
		trace.WithUserIDExtractor(func(c *gin.Context) string {
			auth := c.GetHeader("Authorization")
			if auth == "" {
				return ""
			}
			// 간단한 JWT 토큰 파싱 (실제로는 JWT 라이브러리 사용 권장)
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				// 여기서는 간단히 토큰의 첫 부분을 사용자 ID로 사용
				parts := strings.Split(token, ".")
				if len(parts) > 0 {
					return parts[0]
				}
			}
			return ""
		}),
		trace.WithTokenExtractor(func(c *gin.Context) string {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				return strings.TrimPrefix(auth, "Bearer ")
			}
			return ""
		}),
		trace.WithFilter(func(c *gin.Context) bool {
			// 특정 경로만 추적
			return strings.HasPrefix(c.Request.URL.Path, "/api/")
		}),
	)

	r.GET("/api/jwt", jwtMiddleware, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "JWT-based middleware",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 예제 4: 커스텀 Trace ID 생성
	customTraceMiddleware := trace.MiddlewareWithConfig(
		trace.WithTraceIDGenerator(func(userID, token string) string {
			// UUID 스타일의 Trace ID 생성
			return fmt.Sprintf("trace-%s-%d", userID, time.Now().UnixNano())
		}),
	)

	r.GET("/custom", customTraceMiddleware, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Custom trace ID middleware",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 예제 5: 조건부 필터링
	filteredMiddleware := trace.MiddlewareWithConfig(
		trace.WithFilter(func(c *gin.Context) bool {
			// 성능이 중요한 엔드포인트만 추적
			importantPaths := []string{"/api/users", "/api/orders", "/api/payments"}
			for _, path := range importantPaths {
				if strings.HasPrefix(c.Request.URL.Path, path) {
					return true
				}
			}
			return false
		}),
	)

	r.GET("/api/users", filteredMiddleware, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Filtered middleware - users",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	r.GET("/api/orders", filteredMiddleware, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Filtered middleware - orders",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 예제 6: 쿼리 파라미터 기반 (기본)
	r.GET("/query", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Query parameter middleware",
			"trace_id": c.GetString("trace_id"),
			"user_id":  c.GetString("user_id"),
		})
	})

	// 상태 확인 엔드포인트
	r.GET("/stats", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Trace module is running",
		})
	})

	log.Println("Server starting on :8080")
	log.Println("=== 테스트 URL들 ===")
	log.Println("기본 (쿼리 파라미터): http://localhost:8080/query?user_id=123&access_token=abc123")
	log.Println("헤더 기반: curl -H 'X-User-ID: 123' -H 'X-Access-Token: abc123' http://localhost:8080/header")
	log.Println("JWT 기반: curl -H 'Authorization: Bearer user123.eyJ0b2tlbiI6ImFiYzEyMyJ9.signature' http://localhost:8080/api/jwt")
	log.Println("커스텀 Trace ID: http://localhost:8080/custom?user_id=123&access_token=abc123")
	log.Println("필터링된 API: http://localhost:8080/api/users?user_id=123&access_token=abc123")
	log.Println("상태: http://localhost:8080/stats")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
