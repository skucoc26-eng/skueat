package handler

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"restaurant-api/internal/config"
)

// PageHandler 메인 SPA HTML 페이지 서빙 핸들러
type PageHandler struct {
	cfg *config.Config
}

// NewPageHandler PageHandler 생성
func NewPageHandler(cfg *config.Config) *PageHandler {
	return &PageHandler{cfg: cfg}
}

// Index 메인 페이지 렌더링 (GET /)
func (h *PageHandler) Index(c *gin.Context) {
	session := sessions.Default(c)
	userName, isLoggedIn := getValidUserName(session)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"ApiKey":     h.cfg.KakaoAPIKey,
		"IsLoggedIn": isLoggedIn,
		"UserName":   userName,
		"AppDomain":  h.cfg.AppDomain,
	})
}
