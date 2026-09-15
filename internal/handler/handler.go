package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"restaurant-api/internal/config"
)

// Handler 모든 HTTP 서브 핸들러를 집약한 통합 핸들러 구조체
type Handler struct {
	cfg        *config.Config
	Page       *PageHandler
	Restaurant *RestaurantHandler
	Review     *ReviewHandler
	Auth       *AuthHandler
}

// NewHandler 통합 Handler 생성
func NewHandler(
	cfg *config.Config,
	page *PageHandler,
	rest *RestaurantHandler,
	rev *ReviewHandler,
	auth *AuthHandler,
) *Handler {
	return &Handler{
		cfg:        cfg,
		Page:       page,
		Restaurant: rest,
		Review:     rev,
		Auth:       auth,
	}
}

// SetupRouter 미들웨어, 정적 파일 서빙, 세션 및 라우트를 등록합니다.
func (h *Handler) SetupRouter(r *gin.Engine) {
	// 1. 세션 미들웨어 설정
	store := cookie.NewStore([]byte(h.cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   config.SessionDuration,
		HttpOnly: true,
		Secure:   h.cfg.IsHTTPS(),
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("mysession", store))

	// 2. 정적 파일 캐시 무효화 헤더 미들웨어
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-cache, must-revalidate")
		}
		c.Next()
	})

	// 3. 정적 파일 및 템플릿 로드
	r.Static("/static", "./static")
	r.LoadHTMLGlob("index.html")

	// 4. 라우트 등록
	r.GET("/", h.Page.Index)

	api := r.Group("/api")
	{
		api.GET("/restaurants", h.Restaurant.GetRestaurants)
		api.GET("/restaurants/random", h.Restaurant.GetRandomRestaurant)
		api.GET("/reviews", h.Review.GetReviews)
		api.POST("/rate", h.Review.AddReview)
	}

	auth := r.Group("")
	{
		auth.GET("/login/kakao", h.Auth.LoginKakao)
		auth.GET("/auth/kakao/callback", h.Auth.AuthKakaoCallback)
		auth.GET("/logout", h.Auth.Logout)
	}
}

// getValidUserName 세션에서 사용자명을 조회하고 유효기간을 검증합니다.
func getValidUserName(session sessions.Session) (string, bool) {
	userNameVal := session.Get("userName")
	if userNameVal == nil {
		return "", false
	}
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		return "", false
	}

	loginTimeVal := session.Get("loginTime")
	if loginTimeVal != nil {
		if loginTime, ok := loginTimeVal.(int64); ok {
			if time.Now().Unix()-loginTime > config.SessionDuration {
				session.Clear()
				_ = session.Save()
				return "", false
			}
		}
	}
	return userName, true
}
