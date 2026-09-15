package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// SessionDuration 세션 쿠키 유효 기간 (7일, 초 단위)
const SessionDuration = 7 * 24 * 3600

// Config 애플리케이션 전역 환경 설정 구조체
type Config struct {
	KakaoAPIKey   string
	RestAPIKey    string
	AppDomain     string
	AppHost       string
	AppPort       string
	SessionSecret string
	DatabasePath  string
}

// LoadConfig .env 파일을 읽어들여 Config 구조체로 반환
func LoadConfig() *Config {
	_ = godotenv.Load()

	appDomain := os.Getenv("APP_DOMAIN")
	if appDomain == "" {
		appDomain = "http://localhost:8080"
	}

	appHost := os.Getenv("APP_HOST")
	if appHost == "" {
		appHost = "0.0.0.0"
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "secret"
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "restaurants.db"
	}

	return &Config{
		KakaoAPIKey:   os.Getenv("KAKAO_API_KEY"),
		RestAPIKey:    os.Getenv("REST_API_KEY"),
		AppDomain:     appDomain,
		AppHost:       appHost,
		AppPort:       appPort,
		SessionSecret: sessionSecret,
		DatabasePath:  dbPath,
	}
}

// Addr HTTP 리스닝 주소 반환 (예: 0.0.0.0:8080)
func (c *Config) Addr() string {
	return c.AppHost + ":" + c.AppPort
}

// IsHTTPS 도메인이 HTTPS인지 확인
func (c *Config) IsHTTPS() bool {
	return strings.HasPrefix(c.AppDomain, "https://")
}
