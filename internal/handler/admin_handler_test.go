package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"restaurant-api/internal/config"
	"restaurant-api/internal/model"
)

type adminFixture struct {
	router     *gin.Engine
	cfg        *config.Config
	db         *gorm.DB
	admin      *AdminHandler
	cookies    []*http.Cookie
	csrf       string
	remoteAddr string
	headers    map[string]string
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()
	t.Setenv("FLY_APP_NAME", "")
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}, &model.ReviewModeration{}, &model.UsageEvent{}, &model.AdminSession{}); err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{SessionSecret: "test-secret", AppDomain: "http://example.test", AdminUsername: "owner", AdminPasswordHash: string(hash)}
	r := gin.New()
	t.Chdir("../../cmd")
	h := NewHandler(cfg, NewPageHandler(cfg), NewRestaurantHandler(nil), NewReviewHandler(nil), NewAuthHandler(nil))
	h.Admin = NewAdminHandler(cfg, db)
	h.SetupRouter(r)
	r.GET("/expire-test", func(c *gin.Context) {
		s := sessions.Default(c)
		s.Set("adminUntil", time.Now().Add(-time.Minute).Unix())
		_ = s.Save()
		c.Status(204)
	})
	f := &adminFixture{router: r, cfg: cfg, db: db, admin: h.Admin}
	res := f.request("GET", "/api/admin/session", "", "")
	var v struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	f.csrf = v.CSRF
	return f
}
func (f *adminFixture) request(method, path, body, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if f.remoteAddr != "" {
		r.RemoteAddr = f.remoteAddr
	}
	for name, value := range f.headers {
		r.Header.Set(name, value)
	}
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if f.csrf != "" {
		r.Header.Set("X-CSRF-Token", f.csrf)
	}
	for _, c := range f.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	if cookies := w.Result().Cookies(); len(cookies) > 0 {
		for _, c := range cookies {
			found := false
			for i, old := range f.cookies {
				if c.Name == old.Name {
					f.cookies[i] = c
					found = true
				}
			}
			if !found {
				f.cookies = append(f.cookies, c)
			}
		}
	}
	return w
}

func newAdminVisitor(t *testing.T, server *adminFixture, remoteAddr string) *adminFixture {
	t.Helper()
	f := &adminFixture{router: server.router, cfg: server.cfg, db: server.db, remoteAddr: remoteAddr}
	w := f.request("GET", "/api/admin/session", "", "")
	var data struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	f.csrf = data.CSRF
	return f
}

func TestPublicLogoutRevokesAdminSession(t *testing.T) {
	f := newAdminFixture(t)
	f.login(t)
	copiedCookies := append([]*http.Cookie(nil), f.cookies...)
	if w := f.request("GET", "/logout", "", ""); w.Code != 302 || w.Header().Get("Location") != "/" {
		t.Fatalf("logout=%d %s", w.Code, w.Body.String())
	}
	f.cookies = copiedCookies
	if w := f.request("GET", "/api/admin/stats", "", ""); w.Code != 401 {
		t.Fatalf("copied admin cookie after public logout=%d, want 401", w.Code)
	}
	var count int64
	f.db.Model(&model.AdminSession{}).Count(&count)
	if count != 0 {
		t.Fatalf("admin sessions after logout=%d", count)
	}
}

func TestPublicLogoutDoesNotReportSuccessWhenRevocationFails(t *testing.T) {
	f := newAdminFixture(t)
	f.login(t)
	if err := f.db.Callback().Delete().Before("gorm:delete").Register("fail_admin_revocation", func(tx *gorm.DB) {
		if tx.Statement.Table == "admin_sessions" {
			tx.AddError(errors.New("test DB failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if w := f.request("GET", "/logout", "", ""); w.Code != 500 || w.Header().Get("Location") != "" {
		t.Fatalf("failed revocation reported success: %d", w.Code)
	}
}

func TestAdminLoginFailuresDoNotBlockOtherClients(t *testing.T) {
	for _, fly := range []bool{false, true} {
		t.Run(fmt.Sprint("fly=", fly), func(t *testing.T) {
			attacker := newAdminFixture(t)
			attacker.remoteAddr = "192.0.2.1:1200"
			owner := newAdminVisitor(t, attacker, "192.0.2.2:1300")
			if fly {
				t.Setenv("FLY_APP_NAME", "skueat")
				attacker.remoteAddr, owner.remoteAddr = "127.0.0.1:1200", "127.0.0.1:1300"
				attacker.headers = map[string]string{"Fly-Client-IP": "192.0.2.1"}
				owner.headers = map[string]string{"Fly-Client-IP": "192.0.2.2"}
			}
			for i := 0; i < 10; i++ {
				if w := attacker.request("POST", "/api/admin/login", `{"username":"owner","password":"wrong"}`, "http://example.test"); w.Code != 401 {
					t.Fatalf("failure %d=%d", i, w.Code)
				}
			}
			owner.login(t)
			if w := attacker.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test"); w.Code != 429 {
				t.Fatalf("other client's success reset attacker limit=%d", w.Code)
			}
		})
	}
}

func TestAdminLoginLimitSurvivesCookieAndUntrustedHeaderChanges(t *testing.T) {
	f := newAdminFixture(t)
	f.remoteAddr = "192.0.2.1:1200"
	for i := 0; i < 10; i++ {
		f.request("POST", "/api/admin/login", `{"username":"owner","password":"wrong"}`, "http://example.test")
	}
	retry := newAdminVisitor(t, f, "192.0.2.1:1300")
	retry.headers = map[string]string{"X-Forwarded-For": "192.0.2.2", "Fly-Client-IP": "192.0.2.2"}
	if w := retry.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test"); w.Code != 429 {
		t.Fatalf("cookie/header change bypassed limit=%d", w.Code)
	}
}

func TestAdminLoginLimitExpiresAndStaysBounded(t *testing.T) {
	f := newAdminFixture(t)
	f.remoteAddr = "192.0.2.1:1200"
	now := time.Now()
	for i := 0; i < 1024; i++ {
		f.admin.attempts[fmt.Sprint(i)] = loginFailures{count: 10, since: now}
	}
	if w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test"); w.Code != 429 {
		t.Fatalf("full limiter must fail closed=%d", w.Code)
	}
	if len(f.admin.attempts) != 1024 {
		t.Fatalf("limiter grew to %d", len(f.admin.attempts))
	}
	for ip, attempt := range f.admin.attempts {
		attempt.since = now.Add(-time.Minute)
		f.admin.attempts[ip] = attempt
	}
	f.admin.attempts["192.0.2.1"] = loginFailures{count: 10, since: now.Add(-time.Minute)}
	f.login(t)
	if len(f.admin.attempts) != 0 {
		t.Fatalf("expired entries retained=%d", len(f.admin.attempts))
	}
}

func TestAdminLoginInvalidFlyHeaderCannotResetLimit(t *testing.T) {
	f := newAdminFixture(t)
	t.Setenv("FLY_APP_NAME", "skueat")
	f.remoteAddr = "192.0.2.1:1200"
	f.headers = map[string]string{"Fly-Client-IP": "invalid"}
	for i := 0; i < 10; i++ {
		f.request("POST", "/api/admin/login", `{"username":"owner","password":"wrong"}`, "http://example.test")
	}
	f.headers["Fly-Client-IP"] = "still-invalid"
	if w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test"); w.Code != 429 {
		t.Fatalf("invalid Fly header bypassed socket limit=%d", w.Code)
	}
}
func (f *adminFixture) login(t *testing.T) {
	t.Helper()
	w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test")
	if w.Code != 200 {
		t.Fatalf("login=%d %s", w.Code, w.Body.String())
	}
	var v struct {
		CSRF string `json:"csrf"`
	}
	json.Unmarshal(w.Body.Bytes(), &v)
	f.csrf = v.CSRF
}
func TestAdminAuthorizationCSRFAndRevocation(t *testing.T) {
	f := newAdminFixture(t)
	if w := f.request("GET", "/api/admin/stats", "", ""); w.Code != 401 {
		t.Fatalf("anonymous stats=%d", w.Code)
	}
	if w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test.evil"); w.Code != 403 {
		t.Fatalf("cross-origin login=%d", w.Code)
	}
	token := f.csrf
	f.csrf = "wrong"
	if w := f.request("POST", "/api/admin/login", `{}`, "http://example.test"); w.Code != 403 {
		t.Fatal("bad CSRF accepted")
	}
	f.csrf = token
	f.login(t)
	if w := f.request("GET", "/api/admin/stats?days=7", "", ""); w.Code != 200 {
		t.Fatalf("stats=%d %s", w.Code, w.Body.String())
	}
	if w := f.request("GET", "/api/admin/stats?days=1000", "", ""); w.Code != 400 {
		t.Fatal("unbounded range accepted")
	}
	f.request("GET", "/expire-test", "", "")
	if w := f.request("GET", "/api/admin/stats", "", ""); w.Code != 401 {
		t.Fatal("expired admin accepted")
	}
	f.login(t)
	f.cfg.AdminPasswordHash = ""
	if w := f.request("GET", "/api/admin/reviews", "", ""); w.Code != 401 {
		t.Fatal("disabled admin accepted")
	}
}
func TestAdminModerationAndPrivateFields(t *testing.T) {
	f := newAdminFixture(t)
	r := model.Restaurant{Title: "후로"}
	f.db.Create(&r)
	u := int64(987)
	rating := model.Rating{RestaurantID: r.ID, Score: 1, UserID: "secret-user", KakaoUserID: &u, Comment: "<script>alert(1)</script>"}
	f.db.Create(&rating)
	if w := f.request("POST", "/api/admin/reviews/1/moderation", `{"hidden":true,"reason":"스팸"}`, "http://example.test"); w.Code != 401 {
		t.Fatal("anonymous moderation accepted")
	}
	f.login(t)
	if w := f.request("POST", "/api/admin/reviews/1/moderation", `{"hidden":true}`, "http://example.test"); w.Code != 400 {
		t.Fatal("missing reason accepted")
	}
	if w := f.request("POST", "/api/admin/reviews/1/moderation", `{"hidden":true,"reason":"스팸"}`, "http://example.test"); w.Code != 204 {
		t.Fatalf("moderation=%d %s", w.Code, w.Body.String())
	}
	w := f.request("GET", "/api/admin/reviews?status=hidden", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret-user") || strings.Contains(w.Body.String(), "kakao_user_id") {
		t.Fatal("private identity exposed")
	}
	copiedCookies := append([]*http.Cookie(nil), f.cookies...)
	if w := f.request("POST", "/api/admin/logout", `{}`, "http://example.test"); w.Code != 204 {
		t.Fatal("logout failed")
	}
	if w := f.request("GET", "/api/admin/audit", "", ""); w.Code != 401 {
		t.Fatal("logout not enforced")
	}
	f.cookies = copiedCookies
	if w := f.request("GET", "/api/admin/audit", "", ""); w.Code != 401 {
		t.Fatal("copied session still works after logout")
	}
}
func TestAdminLoginThrottling(t *testing.T) {
	f := newAdminFixture(t)
	for i := 0; i < 10; i++ {
		if w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"wrong"}`, "http://example.test"); w.Code != 401 {
			t.Fatalf("attempt %d=%d", i, w.Code)
		}
	}
	if w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test"); w.Code != 429 {
		t.Fatal("login throttle missing")
	}
}
func TestAnonymousAnalyticsAndAdminExclusion(t *testing.T) {
	f := newAdminFixture(t)
	for i := 0; i < 2; i++ {
		if w := f.request("POST", "/api/events", `{"type":"page_view"}`, "http://example.test"); w.Code != 204 {
			t.Fatalf("event=%d %s", w.Code, w.Body.String())
		}
	}
	if w := f.request("POST", "/api/events", `{"type":"page_view"}`, "http://evil.test"); w.Code != 403 {
		t.Fatal("foreign event accepted")
	}
	if w := f.request("POST", "/api/events", `{"type":"detail","restaurant_id":999}`, "http://example.test"); w.Code != 400 {
		t.Fatal("invalid restaurant accepted")
	}
	var events []model.UsageEvent
	f.db.Find(&events)
	if len(events) != 2 || events[0].Visitor != events[1].Visitor || events[0].Visitor == "" {
		t.Fatal("visitor not stable")
	}
	f.login(t)
	f.request("POST", "/api/events", `{"type":"page_view"}`, "http://example.test")
	var count int64
	f.db.Model(&model.UsageEvent{}).Count(&count)
	if count != 2 {
		t.Fatal("admin visit counted")
	}
}

func TestPublicLogoutWithoutAdminClearsOrdinarySession(t *testing.T) {
	t.Chdir("../../cmd")
	cfg := &config.Config{SessionSecret: "test-secret"}
	r := gin.New()
	h := NewHandler(cfg, NewPageHandler(cfg), NewRestaurantHandler(nil), NewReviewHandler(nil), NewAuthHandler(nil))
	h.SetupRouter(r)
	r.GET("/ordinary-login-test", func(c *gin.Context) {
		s := sessions.Default(c)
		s.Set("userName", "테스터")
		s.Set("kakaoUserID", "123")
		s.Set("loginTime", time.Now().Unix())
		if err := s.Save(); err != nil {
			t.Fatal(err)
		}
		c.Status(204)
	})
	r.GET("/ordinary-user-test", func(c *gin.Context) {
		if _, _, valid := getValidUser(sessions.Default(c)); !valid {
			c.Status(401)
			return
		}
		c.Status(200)
	})
	f := &adminFixture{router: r}
	f.request("GET", "/ordinary-login-test", "", "")
	if w := f.request("GET", "/ordinary-user-test", "", ""); w.Code != 200 {
		t.Fatal("ordinary login failed")
	}
	if w := f.request("GET", "/logout", "", ""); w.Code != 302 {
		t.Fatalf("ordinary logout=%d", w.Code)
	}
	if w := f.request("GET", "/ordinary-user-test", "", ""); w.Code != 401 {
		t.Fatal("ordinary session survived logout")
	}
}
