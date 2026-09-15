package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"restaurant-api/internal/service"
)

// AuthHandler 카카오 OAuth 및 세션 제어 HTTP 핸들러
type AuthHandler struct {
	svc service.AuthService
}

// NewAuthHandler AuthHandler 생성
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// LoginKakao 카카오 간편 로그인 시작 엔드포인트 (GET /login/kakao)
func (h *AuthHandler) LoginKakao(c *gin.Context) {
	authURL := h.svc.GetAuthURL()
	c.Redirect(http.StatusFound, authURL)
}

// AuthKakaoCallback 카카오 인가 코드 수신 및 세션 저장 (GET /auth/kakao/callback)
func (h *AuthHandler) AuthKakaoCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.String(http.StatusBadRequest, "인가 코드가 없습니다.")
		return
	}

	tokenRes, err := h.svc.GetToken(code)
	if err != nil {
		log.Printf("ERROR 카카오 토큰 발급 실패: %v", err)
		c.String(http.StatusInternalServerError, "토큰 발급 실패")
		return
	}

	userInfo, err := h.svc.GetUserInfo(tokenRes.AccessToken)
	if err != nil {
		log.Printf("ERROR 카카오 사용자 정보 조회 실패: %v", err)
		c.String(http.StatusInternalServerError, "사용자 정보 조회 실패")
		return
	}

	session := sessions.Default(c)
	session.Set("userName", userInfo.Properties.Nickname)
	session.Set("loginTime", time.Now().Unix())
	if err := session.Save(); err != nil {
		log.Printf("ERROR 세션 저장 실패: %v", err)
		c.String(http.StatusInternalServerError, "세션 저장 실패")
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// Logout 세션 파기 및 로그아웃 (GET /logout)
func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	_ = session.Save()
	c.Redirect(http.StatusFound, "/")
}
