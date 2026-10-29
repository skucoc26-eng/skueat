package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"restaurant-api/internal/config"
	"restaurant-api/internal/model"
	"restaurant-api/internal/service"
)

//go:embed admin.html
var adminHTML string

type AdminHandler struct {
	cfg      *config.Config
	svc      *service.ManagementService
	db       *gorm.DB
	mu       sync.Mutex
	failures int
	window   time.Time
}

func NewAdminHandler(cfg *config.Config, db *gorm.DB) *AdminHandler {
	return &AdminHandler{cfg: cfg, svc: service.NewManagementService(db), db: db}
}
func (h *AdminHandler) enabled() bool {
	_, err := bcrypt.Cost([]byte(h.cfg.AdminPasswordHash))
	return err == nil
}
func (h *AdminHandler) username() string {
	if h.cfg.AdminUsername != "" {
		return h.cfg.AdminUsername
	}
	return "admin"
}
func randomToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func (h *AdminHandler) version() string {
	sum := sha256.Sum256([]byte(h.username() + ":" + h.cfg.AdminPasswordHash))
	return hex.EncodeToString(sum[:])
}
func (h *AdminHandler) authenticated(c *gin.Context) bool {
	if !h.enabled() {
		return false
	}
	s := sessions.Default(c)
	until, ok := s.Get("adminUntil").(int64)
	v, _ := s.Get("adminVersion").(string)
	if !ok || until <= time.Now().Unix() || until > time.Now().Add(4*time.Hour+time.Minute).Unix() || !hmac.Equal([]byte(v), []byte(h.version())) {
		return false
	}
	token, _ := s.Get("adminToken").(string)
	if token == "" {
		return false
	}
	var count int64
	return h.db.Model(&model.AdminSession{}).Where("token = ? AND expires_at > ?", token, time.Now().UTC()).Count(&count).Error == nil && count == 1
}
func (h *AdminHandler) requireAdmin(c *gin.Context) {
	if !h.authenticated(c) {
		c.AbortWithStatusJSON(401, gin.H{"error": "관리자 로그인이 필요합니다."})
		return
	}
	c.Next()
}
func (h *AdminHandler) sameOrigin(c *gin.Context) bool {
	u, err := url.Parse(c.GetHeader("Origin"))
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	expected, err := url.Parse(h.cfg.AppDomain)
	return err == nil && u.Scheme == expected.Scheme && u.Host == expected.Host
}
func (h *AdminHandler) csrf(c *gin.Context) {
	expected, _ := sessions.Default(c).Get("adminCSRF").(string)
	provided := c.GetHeader("X-CSRF-Token")
	if !h.sameOrigin(c) || expected == "" || !hmac.Equal([]byte(expected), []byte(provided)) {
		c.AbortWithStatusJSON(403, gin.H{"error": "요청을 확인할 수 없습니다. 새로고침 후 다시 시도해 주세요."})
		return
	}
	c.Next()
}
func (h *AdminHandler) Register(r *gin.Engine) {
	r.GET("/admin", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		c.Data(200, "text/html; charset=utf-8", []byte(adminHTML))
	})
	g := r.Group("/api/admin")
	g.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	g.GET("/session", h.Session)
	g.POST("/login", h.csrf, h.Login)
	protected := g.Group("")
	protected.Use(h.requireAdmin)
	protected.POST("/logout", h.csrf, h.Logout)
	protected.GET("/stats", h.Stats)
	protected.GET("/reviews", h.Reviews)
	protected.GET("/audit", h.Audit)
	protected.POST("/reviews/:id/moderation", h.csrf, h.Moderate)
	r.POST("/api/events", h.Event)
}
func (h *AdminHandler) Session(c *gin.Context) {
	s := sessions.Default(c)
	csrf, _ := s.Get("adminCSRF").(string)
	if csrf == "" {
		csrf = randomToken()
		s.Set("adminCSRF", csrf)
		if s.Save() != nil {
			c.Status(500)
			return
		}
	}
	c.JSON(200, gin.H{"enabled": h.enabled(), "authenticated": h.authenticated(c), "csrf": csrf, "username": h.username()})
}
func (h *AdminHandler) Login(c *gin.Context) {
	if !h.enabled() {
		c.JSON(503, gin.H{"error": "관리자 계정이 아직 설정되지 않았습니다."})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if time.Since(h.window) >= time.Minute {
		h.window = time.Now()
		h.failures = 0
	}
	if h.failures >= 10 {
		c.Header("Retry-After", "60")
		c.JSON(429, gin.H{"error": "로그인 시도가 많습니다. 1분 뒤 다시 시도해 주세요."})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&input) != nil || len(input.Password) > 72 {
		c.JSON(400, gin.H{"error": "입력값을 확인해 주세요."})
		return
	}
	validPassword := bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(input.Password)) == nil
	validUser := subtle.ConstantTimeCompare([]byte(input.Username), []byte(h.username())) == 1
	if !validUser || !validPassword {
		h.failures++
		c.JSON(401, gin.H{"error": "아이디 또는 비밀번호가 일치하지 않습니다."})
		return
	}
	h.failures = 0
	s := sessions.Default(c)
	if err := h.db.Where("expires_at <= ?", time.Now().UTC()).Delete(&model.AdminSession{}).Error; err != nil {
		c.Status(500)
		return
	}
	if old, ok := s.Get("adminToken").(string); ok {
		if err := h.db.Where("token = ?", old).Delete(&model.AdminSession{}).Error; err != nil {
			c.Status(500)
			return
		}
	}
	token := randomToken()
	if err := h.db.Create(&model.AdminSession{Token: token, ExpiresAt: time.Now().Add(4 * time.Hour).UTC()}).Error; err != nil {
		c.Status(500)
		return
	}
	s.Set("adminToken", token)
	s.Set("adminUntil", time.Now().Add(4*time.Hour).Unix())
	s.Set("adminVersion", h.version())
	s.Set("adminCSRF", randomToken())
	if s.Save() != nil {
		c.Status(500)
		return
	}
	c.JSON(200, gin.H{"csrf": s.Get("adminCSRF")})
}
func (h *AdminHandler) Logout(c *gin.Context) {
	s := sessions.Default(c)
	if token, ok := s.Get("adminToken").(string); ok {
		if err := h.db.Where("token = ?", token).Delete(&model.AdminSession{}).Error; err != nil {
			c.Status(500)
			return
		}
	}
	s.Delete("adminToken")
	s.Delete("adminUntil")
	s.Delete("adminVersion")
	s.Set("adminCSRF", randomToken())
	if s.Save() != nil {
		c.Status(500)
		return
	}
	c.Status(204)
}
func managementError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrManagementInput):
		c.JSON(400, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(404, gin.H{"error": "대상을 찾을 수 없습니다."})
	case errors.Is(err, service.ErrUsageLimit):
		c.JSON(429, gin.H{"error": err.Error()})
	default:
		c.JSON(500, gin.H{"error": "처리 중 오류가 발생했습니다."})
	}
}
func (h *AdminHandler) Stats(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	stats, err := h.svc.Statistics(days, time.Now())
	if err != nil {
		managementError(c, err)
		return
	}
	c.JSON(200, stats)
}
func (h *AdminHandler) Reviews(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	rows, total, err := h.svc.Reviews(c.DefaultQuery("status", "all"), c.Query("search"), page)
	if err != nil {
		managementError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": total, "page": page, "page_size": 50})
}
func (h *AdminHandler) Moderate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		managementError(c, service.ErrManagementInput)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var input struct {
		Hidden *bool  `json:"hidden"`
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&input) != nil || input.Hidden == nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	if err := h.svc.Moderate(uint(id), *input.Hidden, input.Reason, h.username()); err != nil {
		managementError(c, err)
		return
	}
	c.Status(204)
}
func (h *AdminHandler) Audit(c *gin.Context) {
	rows, err := h.svc.Audit()
	if err != nil {
		managementError(c, err)
		return
	}
	c.JSON(200, rows)
}

func (h *AdminHandler) visitor(c *gin.Context) string {
	value, _ := c.Cookie("skueat_visitor")
	parts := strings.Split(value, ".")
	sign := func(token string) string {
		mac := hmac.New(sha256.New, []byte(h.cfg.SessionSecret))
		mac.Write([]byte("visitor:" + token))
		return hex.EncodeToString(mac.Sum(nil))
	}
	if len(parts) == 2 && len(parts[0]) == 64 && hmac.Equal([]byte(sign(parts[0])), []byte(parts[1])) {
		return parts[1]
	}
	token := randomToken()
	signature := sign(token)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("skueat_visitor", token+"."+signature, 90*86400, "/", "", h.cfg.IsHTTPS(), true)
	return signature
}
func (h *AdminHandler) Event(c *gin.Context) {
	if !h.sameOrigin(c) {
		c.Status(403)
		return
	}
	if h.authenticated(c) {
		c.Status(204)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var input struct {
		Type         string `json:"type"`
		RestaurantID uint   `json:"restaurant_id"`
		Query        string `json:"query"`
		Category     string `json:"category"`
		Source       string `json:"source"`
	}
	if c.ShouldBindJSON(&input) != nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	err := h.svc.RecordEvent(model.UsageEvent{Visitor: h.visitor(c), Type: input.Type, RestaurantID: input.RestaurantID, Query: input.Query, Category: input.Category, Source: input.Source}, time.Now())
	if err != nil {
		managementError(c, err)
		return
	}
	c.Status(204)
}
