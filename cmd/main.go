package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"restaurant-api/internal/config"
	"restaurant-api/internal/handler"
	"restaurant-api/internal/repository"
	"restaurant-api/internal/service"
)

func main() {
	// 1. 환경 설정 로드
	cfg := config.LoadConfig()

	// 2. 로깅 시스템 초기화
	setupLogger()

	// 3. 데이터베이스 초기화 (embed된 restaurants.json 데이터 함께 주입)
	seedJSON := getSeedRestaurantsJSON()
	db, err := repository.InitDB(cfg.DatabasePath, seedJSON)
	if err != nil {
		log.Fatalf("FATAL 데이터베이스 초기화 실패: %v", err)
	}
	log.Println("INFO  데이터베이스 연결 및 초기화 완료")

	// 4. 의존성 주입 (Repository -> Service -> Handler)
	restRepo := repository.NewRestaurantRepository(db)
	revRepo := repository.NewReviewRepository(db)

	restSvc := service.NewRestaurantService(restRepo)
	revSvc := service.NewReviewService(db, restRepo, revRepo)
	authSvc := service.NewAuthService(cfg)

	pageHandler := handler.NewPageHandler(cfg)
	restHandler := handler.NewRestaurantHandler(restSvc)
	revHandler := handler.NewReviewHandler(revSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	appHandler := handler.NewHandler(cfg, pageHandler, restHandler, revHandler, authHandler)

	// 5. Gin 엔진 생성 및 미들웨어/라우터 구성
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(customLoggerMiddleware())

	staticFS, err := getStaticFS()
	if err != nil {
		log.Printf("WARN  내장 static FS 로드 실패: %v", err)
	}
	tmpl, err := getIndexTemplate()
	if err != nil {
		log.Printf("WARN  내장 index.html 로드 실패: %v", err)
	}

	appHandler.SetupRouter(r, staticFS, tmpl)

	// 6. 서버 실행
	log.Printf("INFO  서버가 %s 에서 리스닝 중입니다.\n", cfg.Addr())
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("FATAL 서버 실행 실패: %v", err)
	}
}

// setupLogger 로그 디렉토리 및 파일/표준출력 멀티라이터 설정
func setupLogger() {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		_ = os.Mkdir("logs", 0755)
	}

	f, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("로그 파일을 열 수 없습니다: %v\n", err)
		return
	}

	multiWriter := io.MultiWriter(f, os.Stdout)
	gin.DefaultWriter = multiWriter
	log.SetOutput(multiWriter)
	log.SetFlags(0)
}

// customLoggerMiddleware Gin 커스텀 로그 포맷터
func customLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		level := "INFO "
		if param.StatusCode >= 400 && param.StatusCode < 500 {
			level = "WARN "
		} else if param.StatusCode >= 500 {
			level = "ERROR"
		}

		return fmt.Sprintf("%s [%d] %s %s (%s)\n",
			level,
			param.StatusCode,
			param.Method,
			param.Path,
			param.Latency,
		)
	})
}
