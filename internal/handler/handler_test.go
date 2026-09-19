package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestLegacySessionRequiresLogin(t *testing.T) {
	for _, id := range []interface{}{nil, "", 123, "kakao_123"} {
		r := gin.New()
		r.Use(sessions.Sessions("test", cookie.NewStore([]byte("test-secret"))))
		r.GET("/", func(c *gin.Context) {
			s := sessions.Default(c)
			s.Set("userName", "홍길동")
			s.Set("loginTime", time.Now().Unix())
			if id != nil {
				s.Set("userID", id)
			}
			uid, _, ok := getValidUser(s)
			if id == "kakao_123" {
				if !ok || uid != id {
					t.Fatal("authenticated account rejected")
				}
			} else {
				if ok || s.Get("userName") != nil {
					t.Fatalf("legacy session accepted or retained: %v", id)
				}
			}
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}
}
