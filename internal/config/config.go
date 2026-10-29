package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// SessionDuration 세션 쿠키 유효 기간 (7일, 초 단위)
const SessionDuration = 7 * 24 * 3600

// Config 애플리케이션 전역 환경 설정 구조체
type Config struct {
	KakaoAPIKey       string
	RestAPIKey        string
	AppDomain         string
	AppHost           string
	AppPort           string
	SessionSecret     string
	DatabasePath      string
	AdminUsername     string
	AdminPasswordHash string
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
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("FATAL 세션 비밀키 생성 실패: %v", err)
		}
		sessionSecret = hex.EncodeToString(key)
		log.Println("WARN  SESSION_SECRET 환경변수가 비어있어 임시 세션 키를 생성했습니다. 재시작 시 기존 로그인 세션이 만료됩니다.")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "restaurants.db"
	}

	return &Config{
		KakaoAPIKey:       os.Getenv("KAKAO_API_KEY"),
		RestAPIKey:        os.Getenv("REST_API_KEY"),
		AppDomain:         appDomain,
		AppHost:           appHost,
		AppPort:           appPort,
		SessionSecret:     sessionSecret,
		DatabasePath:      dbPath,
		AdminUsername:     os.Getenv("ADMIN_USERNAME"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
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
