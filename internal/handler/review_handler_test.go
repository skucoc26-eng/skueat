package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
	"restaurant-api/internal/service"
)

func TestRateEndpointValidationAndPublicReview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}); err != nil {
		t.Fatal(err)
	}
	restaurant := model.Restaurant{Title: "테스트 식당"}
	if err := db.Create(&restaurant).Error; err != nil {
		t.Fatal(err)
	}
	restRepo := repository.NewRestaurantRepository(db)
	h := NewReviewHandler(service.NewReviewService(db, restRepo, repository.NewReviewRepository(db)))
	router := gin.New()
	router.Use(sessions.Sessions("mysession", cookie.NewStore([]byte("test-secret"))))
	router.GET("/test-login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("userName", "테스터")
		session.Set("loginTime", time.Now().Unix())
		if err := session.Save(); err != nil {
			t.Fatal(err)
		}
		c.Status(http.StatusNoContent)
	})
	router.POST("/api/rate", h.AddReview)
	router.GET("/api/reviews", h.GetReviews)

	post := func(values url.Values, cookies []*http.Cookie) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/rate", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		router.ServeHTTP(rec, req)
		return rec
	}
	form := url.Values{"restaurant_id": {"1"}, "score": {"2"}, "comment": {"맛있어요"}}
	if got := post(form, nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated rate status=%d, want 401", got.Code)
	}
	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/test-login", nil))
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("test login did not set session cookie")
	}
	for _, invalid := range []url.Values{
		{"restaurant_id": {"1"}, "score": {"6"}},
		{"restaurant_id": {"1"}, "score": {"2"}, "comment": {strings.Repeat("x", 301)}},
	} {
		if got := post(invalid, cookies); got.Code != http.StatusBadRequest {
			t.Errorf("invalid rate status=%d, want 400", got.Code)
		}
	}
	var count int64
	db.Model(&model.Rating{}).Count(&count)
	if count != 0 {
		t.Fatalf("invalid requests persisted %d ratings, want 0", count)
	}
	if got := post(form, cookies); got.Code != http.StatusOK {
		t.Fatalf("valid rate status=%d, body=%s", got.Code, got.Body.String())
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/reviews?restaurant_id=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("reviews status=%d", rec.Code)
	}
	var reviews []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &reviews); err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || reviews[0]["author_name"] != "테*터" {
		t.Fatalf("public reviews=%v, want one masked review", reviews)
	}
	if _, exposed := reviews[0]["user_id"]; exposed {
		t.Fatal("public review contains user_id")
	}
}
