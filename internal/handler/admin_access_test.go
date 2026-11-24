package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"restaurant-api/internal/model"
)

func ownerLoginBody(otp string) string {
	return fmt.Sprintf(`{"username":"owner","password":"test-password","otp":%q}`, otp)
}

func TestAdminPageRequiresRole(t *testing.T) {
	f := newAdminFixture(t)
	for _, path := range []string{"/", "/admin"} {
		response := f.request("GET", path, "", "")
		if strings.Contains(response.Body.String(), "운영자 관리") || strings.Contains(response.Body.String(), "스쿠잇 운영 현황") {
			t.Errorf("anonymous response to %s exposes management UI", path)
		}
		if path == "/admin" && response.Code != 404 {
			t.Errorf("/admin status = %d, want 404", response.Code)
		}
	}
	if w := f.request("GET", "/admin/login", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "인증 코드") {
		t.Errorf("separate login response = %d", w.Code)
	}
}

func TestAdminPasswordAloneCannotCreateSession(t *testing.T) {
	f := newAdminFixture(t)
	w := f.request("POST", "/api/admin/login", `{"username":"owner","password":"test-password"}`, "http://example.test")
	if w.Code != 401 {
		t.Fatalf("password-only login posted status %d, want 401", w.Code)
	}
	if w := f.request("GET", "/api/admin/users", "", ""); w.Code != 401 {
		t.Fatal("password-only login created an admin session")
	}
}

func TestAdminTOTPMatchesRFC6238AndAcceptsOnlyRecentSteps(t *testing.T) {
	key := adminTOTPKey("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	// RFC 6238 SHA-1 test code 94287082, reduced to the final six digits.
	for _, tc := range []struct {
		unix   int64
		accept bool
	}{{29, false}, {59, true}, {60, true}, {90, false}} {
		if _, ok := adminTOTPStep(key, "287082", time.Unix(tc.unix, 0)); ok != tc.accept {
			t.Fatalf("at Unix %d, accepted = %v", tc.unix, ok)
		}
	}
}

func TestAdminTOTPRejectsReuseAcrossBrowsers(t *testing.T) {
	owner := newAdminFixture(t)
	now := time.Now().Truncate(30 * time.Second)
	owner.admin.now = func() time.Time { return now }
	other := newAdminVisitor(t, owner, "192.0.2.44:1200")
	code := adminTOTPCode(adminTOTPKey(owner.cfg.AdminTOTPSecret), now.Unix()/30)
	signIn := func(f *adminFixture, otp string, want int) {
		t.Helper()
		if w := f.request("POST", "/api/admin/login", ownerLoginBody(otp), "http://example.test"); w.Code != want {
			t.Fatalf("login status = %d, want %d", w.Code, want)
		}
	}
	signIn(owner, "abcdef", 401)
	signIn(owner, code, 200)
	if w := owner.request("GET", "/admin", "", ""); w.Code != 200 {
		t.Fatalf("owner dashboard = %d", w.Code)
	}
	signIn(other, code, 401)
	now = now.Add(30 * time.Second)
	signIn(other, code, 401)
	signIn(other, adminTOTPCode(adminTOTPKey(owner.cfg.AdminTOTPSecret), now.Unix()/30), 200)
	var sessions int64
	if err := owner.db.Model(&model.AdminSession{}).Count(&sessions).Error; err != nil || sessions != 2 {
		t.Fatalf("admin sessions = %d, error = %v", sessions, err)
	}
	owner.cfg.AdminTOTPSecret = "MFRGGZDFMZTWQ2LKNNWG23TPOBYXE43U"
	if w := owner.request("GET", "/admin", "", ""); w.Code != 404 {
		t.Fatal("key rotation preserved an old session")
	}
	owner.cfg.AdminTOTPSecret = "invalid"
	refreshVisitorCSRF(t, other)
	signIn(other, code, 503)
}
