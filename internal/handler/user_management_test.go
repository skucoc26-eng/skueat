package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gorm.io/gorm"
	"net/http/httptest"
	"restaurant-api/internal/model"
)

func loginKakaoVisitor(t *testing.T, f *adminFixture) *httptest.ResponseRecorder {
	t.Helper()
	start := f.request("GET", "/login/kakao?next=/admin", "", "")
	location, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return f.request("GET", "/auth/kakao/callback?code=test&state="+url.QueryEscape(location.Query().Get("state")), "", "")
}

func refreshVisitorCSRF(t *testing.T, f *adminFixture) map[string]interface{} {
	t.Helper()
	w := f.request("GET", "/api/admin/session", "", "")
	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	f.csrf, _ = data["csrf"].(string)
	return data
}

func TestUserManagementAuthorizationAndImmediateRevocation(t *testing.T) {
	owner := newAdminFixture(t)
	owner.login(t)
	member := newAdminVisitor(t, owner, "192.0.2.42:1200")
	if w := loginKakaoVisitor(t, member); w.Code != 302 || w.Header().Get("Location") != "/admin" {
		t.Fatalf("Kakao login=%d %s", w.Code, w.Body.String())
	}
	refreshVisitorCSRF(t, member)
	var user model.User
	if err := owner.db.First(&user).Error; err != nil {
		t.Fatal(err)
	}
	rolePath := fmt.Sprintf("/api/admin/users/%d/role", user.ID)
	for _, path := range []string{"/api/admin/stats", "/api/admin/users", "/api/admin/user-role-audit"} {
		if w := member.request("GET", path, "", ""); w.Code != 403 {
			t.Fatalf("regular user %s=%d", path, w.Code)
		}
	}
	if w := member.request("POST", rolePath, `{"role":"operator"}`, "http://example.test"); w.Code != 403 {
		t.Fatal("self promotion allowed")
	}
	oldCSRF := owner.csrf
	owner.csrf = "wrong"
	if w := owner.request("POST", rolePath, `{"role":"operator"}`, "http://example.test"); w.Code != 403 {
		t.Fatal("role CSRF accepted")
	}
	owner.csrf = oldCSRF
	if w := owner.request("POST", rolePath, `{"role":"owner"}`, "http://example.test"); w.Code != 400 {
		t.Fatal("root role accepted")
	}
	if w := owner.request("POST", "/api/admin/users/99999/role", `{"role":"operator"}`, "http://example.test"); w.Code != 404 {
		t.Fatal("missing member accepted")
	}
	if w := owner.request("POST", rolePath, `{"role":"operator"}`, "http://example.test"); w.Code != 204 {
		t.Fatalf("grant=%d %s", w.Code, w.Body.String())
	}
	state := refreshVisitorCSRF(t, member)
	if state["authenticated"] != true || state["owner"] != false {
		t.Fatalf("operator session=%+v", state)
	}
	for _, path := range []string{"/api/admin/stats", "/api/admin/reviews", "/api/admin/audit"} {
		if w := member.request("GET", path, "", ""); w.Code != 200 {
			t.Fatalf("operator %s=%d", path, w.Code)
		}
	}
	for _, path := range []string{"/api/admin/users", "/api/admin/user-role-audit"} {
		if w := member.request("GET", path, "", ""); w.Code != 403 {
			t.Fatalf("operator saw users=%d", w.Code)
		}
	}
	if w := member.request("POST", rolePath, `{"role":"user"}`, "http://example.test"); w.Code != 403 {
		t.Fatal("operator changed roles")
	}
	rating := model.Rating{RestaurantID: 1, Score: 2, Comment: "test"}
	owner.db.Create(&model.Restaurant{Title: "식당"})
	owner.db.Create(&rating)
	if w := member.request("POST", fmt.Sprintf("/api/admin/reviews/%d/moderation", rating.ID), `{"hidden":true,"reason":"스팸"}`, "http://example.test"); w.Code != 204 {
		t.Fatalf("operator moderation=%d", w.Code)
	}
	audit, _ := owner.admin.svc.Audit()
	if len(audit) != 1 || audit[0].Admin != fmt.Sprintf("operator:%d", user.ID) {
		t.Fatal("operator audit actor missing")
	}
	w := owner.request("GET", "/api/admin/users?search="+url.QueryEscape(user.Nickname), "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "kakao") || strings.Contains(w.Body.String(), "token") {
		t.Fatalf("private fields in user list: %s", w.Body.String())
	}
	if w := member.request("POST", "/api/events", `{"type":"invalid-admin-check"}`, "http://example.test"); w.Code != 204 {
		t.Fatal("operator activity was counted")
	}
	if w := owner.request("POST", rolePath, `{"role":"user"}`, "http://example.test"); w.Code != 204 {
		t.Fatal("revoke failed")
	}
	if w := member.request("GET", "/api/admin/stats", "", ""); w.Code != 403 {
		t.Fatal("revocation not immediate")
	}
	owner.request("POST", rolePath, `{"role":"operator"}`, "http://example.test")
	if w := member.request("GET", "/api/admin/stats", "", ""); w.Code != 403 {
		t.Fatal("regrant revived revoked token")
	}
	if w := loginKakaoVisitor(t, member); w.Code != 302 {
		t.Fatal("operator re-login failed")
	}
	refreshVisitorCSRF(t, member)
	copied := append([]*http.Cookie(nil), member.cookies...)
	if w := member.request("POST", "/api/admin/logout", `{}`, "http://example.test"); w.Code != 204 {
		t.Fatal("operator logout failed")
	}
	member.cookies = copied
	if w := member.request("GET", "/api/admin/stats", "", ""); w.Code != 403 {
		t.Fatal("copied operator cookie survived logout")
	}
}

func TestKakaoUserStorageFailureDoesNotCompleteLogin(t *testing.T) {
	f := newAdminFixture(t)
	if err := f.db.Callback().Create().Before("gorm:create").Register("reject_user", func(tx *gorm.DB) {
		if tx.Statement.Table == "users" {
			tx.AddError(errors.New("database unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if w := loginKakaoVisitor(t, f); w.Code != 500 {
		t.Fatalf("failed DB login=%d", w.Code)
	}
	var count int64
	f.db.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatal("failed member persisted")
	}
	f.db.Model(&model.AdminSession{}).Count(&count)
	if count != 0 {
		t.Fatal("failed login issued management session")
	}
	state := refreshVisitorCSRF(t, f)
	if state["logged_in"] != false {
		t.Fatal("failed DB login issued user cookie")
	}
}

func TestKakaoReturnPathDoesNotRedirectExternally(t *testing.T) {
	f := newAdminFixture(t)
	w := f.request("GET", "/login/kakao?next=https://evil.test", "", "")
	location, _ := url.Parse(w.Header().Get("Location"))
	w = f.request("GET", "/auth/kakao/callback?code=test&state="+url.QueryEscape(location.Query().Get("state")), "", "")
	if w.Code != 302 || w.Header().Get("Location") != "/" {
		t.Fatal("external OAuth return redirect")
	}
}
