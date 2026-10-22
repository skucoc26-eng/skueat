package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"restaurant-api/internal/model"
)

type stubAuthService struct {
	tokenCalls int
}

func (s *stubAuthService) GetAuthURL(state string) string {
	return "https://example.test/oauth?state=" + url.QueryEscape(state)
}

func (s *stubAuthService) GetToken(string) (*model.KakaoTokenResponse, error) {
	s.tokenCalls++
	return &model.KakaoTokenResponse{AccessToken: "test-token"}, nil
}

func (s *stubAuthService) GetUserInfo(string) (*model.KakaoUserResponse, error) {
	res := &model.KakaoUserResponse{}
	res.Properties.Nickname = "테스터"
	return res, nil
}

func TestKakaoOAuthState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := &stubAuthService{}
	router := gin.New()
	router.Use(sessions.Sessions("mysession", cookie.NewStore([]byte("test-secret"))))
	h := NewAuthHandler(auth)
	router.GET("/login/kakao", h.LoginKakao)
	router.GET("/auth/kakao/callback", h.AuthKakaoCallback)

	login := httptest.NewRecorder()
	router.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/login/kakao", nil))
	if login.Code != http.StatusFound {
		t.Fatalf("login status=%d, want 302", login.Code)
	}
	location, err := url.Parse(login.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := location.Query().Get("state")
	if len(state) != 32 {
		t.Fatalf("state length=%d, want 32 hex characters", len(state))
	}
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not save state in session cookie")
	}
	callback := func(stateValue string, sessionCookies []*http.Cookie) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/auth/kakao/callback?code=test&state="+url.QueryEscape(stateValue), nil)
		for _, c := range sessionCookies {
			req.AddCookie(c)
		}
		router.ServeHTTP(rec, req)
		return rec
	}
	for _, input := range []string{"", "wrong-state"} {
		if got := callback(input, cookies); got.Code != http.StatusBadRequest {
			t.Errorf("state %q returned %d, want 400", input, got.Code)
		}
	}
	if auth.tokenCalls != 0 {
		t.Fatal("invalid state reached token exchange")
	}
	valid := callback(state, cookies)
	if valid.Code != http.StatusFound || !strings.HasSuffix(valid.Header().Get("Location"), "/") {
		t.Fatalf("valid state returned %d, location=%q; want redirect to /", valid.Code, valid.Header().Get("Location"))
	}
	if auth.tokenCalls != 1 {
		t.Fatalf("valid state exchanged token %d times, want 1", auth.tokenCalls)
	}
	if got := callback(state, valid.Result().Cookies()); got.Code != http.StatusBadRequest {
		t.Fatalf("replayed state returned %d, want 400", got.Code)
	}
}
