package handler

import (
	"crypto/rand"
	"encoding/hex"
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
	// CSRF 방지를 위한 암호학적 32바이트 state 난수 생성
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)

	session := sessions.Default(c)
	session.Set("oauth_state", state)
	_ = session.Save()

	authURL := h.svc.GetAuthURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// AuthKakaoCallback 카카오 인가 코드 수신 및 세션 저장 (GET /auth/kakao/callback)
func (h *AuthHandler) AuthKakaoCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.String(http.StatusBadRequest, "인가 코드가 없습니다.")
		return
	}

	state := c.Query("state")
	session := sessions.Default(c)
	savedState := session.Get("oauth_state")

	// CSRF 방어를 위한 state 일치 검증
	if savedState == nil || savedState.(string) != state || state == "" {
		c.String(http.StatusBadRequest, "유효하지 않거나 만료된 요청입니다 (CSRF 검증 실패).")
		return
	}
	session.Delete("oauth_state")

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
