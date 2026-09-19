package handler

import (
	"html/template"
	"io/fs"
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
// staticFS와 tmpl이 전달되면 바이너리에 내장된 embed 자원을 사용하고, 없으면 디스크 파일 시스템을 fallback으로 사용합니다.
func (h *Handler) SetupRouter(r *gin.Engine, staticFS fs.FS, tmpl *template.Template) {
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

	// 3. 정적 파일 및 템플릿 로드 (내장 embed 자원 우선, 없을 경우 디스크 파일 탐색)
	if staticFS != nil {
		r.StaticFS("/static", http.FS(staticFS))
	} else {
		r.Static("/static", "./static")
	}

	if tmpl != nil {
		r.SetHTMLTemplate(tmpl)
	} else {
		r.LoadHTMLGlob("index.html")
	}

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

// getValidUser 세션에서 사용자 식별자(userID)와 이름(userName)을 조회하고 유효기간을 검증합니다.
func getValidUser(session sessions.Session) (string, string, bool) {
	userNameVal := session.Get("userName")
	if userNameVal == nil {
		return "", "", false
	}
	userName, ok := userNameVal.(string)
	if !ok || userName == "" {
		return "", "", false
	}

	loginTimeVal := session.Get("loginTime")
	if loginTimeVal != nil {
		if loginTime, ok := loginTimeVal.(int64); ok {
			if time.Now().Unix()-loginTime > config.SessionDuration {
				session.Clear()
				_ = session.Save()
				return "", "", false
			}
		}
	}

	userID := ""
	if uidVal := session.Get("userID"); uidVal != nil {
		if uid, ok := uidVal.(string); ok {
			userID = uid
		}
	}
	if userID == "" {
		// 닉네임만 저장된 이전 세션은 소유권을 확인할 수 없으므로 재로그인합니다.
		session.Clear()
		_ = session.Save()
		return "", "", false
	}

	return userID, userName, true
}

// getValidUserName 세션에서 사용자명을 조회하고 유효기간을 검증합니다.
func getValidUserName(session sessions.Session) (string, bool) {
	_, userName, ok := getValidUser(session)
	return userName, ok
}
