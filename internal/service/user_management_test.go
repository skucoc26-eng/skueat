package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"restaurant-api/internal/model"
)

func TestUserLoginPreservesIdentityRoleAndRotatesSession(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	now := time.Now().UTC().Truncate(time.Second)
	first, err := s.LoginUser(123, "처음", strings.Repeat("a", 64), "", now)
	if err != nil || first.Role != "user" {
		t.Fatalf("first login: %+v %v", first, err)
	}
	if err := s.SetUserRole(first.ID, "operator", "admin"); err != nil {
		t.Fatal(err)
	}
	next, err := s.LoginUser(123, "새 닉네임", strings.Repeat("b", 64), strings.Repeat("a", 64), now.Add(time.Hour))
	if err != nil || next.ID != first.ID || next.Role != "operator" || !next.CreatedAt.Equal(first.CreatedAt) || !next.LastLoginAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("repeat login changed identity/role: %+v %v", next, err)
	}
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate member")
	}
	db.Model(&model.AdminSession{}).Where("token = ?", strings.Repeat("a", 64)).Count(&count)
	if count != 0 {
		t.Fatal("old browser management token survived")
	}
	db.Model(&model.AdminSession{}).Where("token = ? AND user_id = ?", strings.Repeat("b", 64), first.ID).Count(&count)
	if count != 1 {
		t.Fatal("new management token missing")
	}
	for _, role := range []string{"owner", "admin", "", "OPERATOR"} {
		if err := s.SetUserRole(first.ID, role, "admin"); !errors.Is(err, ErrManagementInput) {
			t.Fatalf("invalid role %q=%v", role, err)
		}
	}
	if _, err := s.LoginUser(0, "bad", strings.Repeat("c", 64), "", now); !errors.Is(err, ErrManagementInput) {
		t.Fatal("invalid identity accepted")
	}
}

func TestUsersListSearchPaginationAndReviewCounts(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	now := time.Now()
	u, err := s.LoginUser(456, "목록 회원", strings.Repeat("a", 64), "", now)
	if err != nil {
		t.Fatal(err)
	}
	id := int64(456)
	for i := 0; i < 2; i++ {
		if err := db.Create(&model.Rating{RestaurantID: uint(i + 1), KakaoUserID: &id, Score: 3, Hidden: i == 1}).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err := s.Users("목록", 1)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ReviewCount != 2 {
		t.Fatalf("counts=%+v %d %v", rows, total, err)
	}
	rows, total, err = s.Users(fmt.Sprint(u.ID), 1)
	if err != nil || total != 1 || rows[0].ID != u.ID {
		t.Fatal("internal ID lookup failed")
	}
	for i := 0; i < 50; i++ {
		if err := db.Create(&model.User{KakaoUserID: int64(1000 + i), Nickname: fmt.Sprint("other", i), CreatedAt: now, LastLoginAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err = s.Users("", 2)
	if err != nil || total != 51 || len(rows) != 1 {
		t.Fatalf("pagination=%d/%d %v", len(rows), total, err)
	}
	for _, page := range []int{0, 10001} {
		if _, _, err := s.Users("", page); !errors.Is(err, ErrManagementInput) {
			t.Fatal("invalid page accepted")
		}
	}
	if _, _, err := s.Users(strings.Repeat("가", 81), 1); !errors.Is(err, ErrManagementInput) {
		t.Fatal("long search accepted")
	}
}

func TestRoleAuditAndSessionRevocationAreAtomic(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	u, err := s.LoginUser(123, "회원", strings.Repeat("a", 64), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserRole(u.ID, "operator", "admin"); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("reject_role_audit", func(tx *gorm.DB) {
		if tx.Statement.Table == "user_role_changes" {
			tx.AddError(errors.New("audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserRole(u.ID, "user", "admin"); err == nil {
		t.Fatal("audit failure ignored")
	}
	db.First(&u, u.ID)
	if u.Role != "operator" {
		t.Fatal("role not rolled back")
	}
	var count int64
	db.Model(&model.AdminSession{}).Where("user_id = ?", u.ID).Count(&count)
	if count != 1 {
		t.Fatal("session not rolled back")
	}
	db.Callback().Create().Remove("reject_role_audit")
	if err := s.SetUserRole(u.ID, "user", "admin"); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.AdminSession{}).Where("user_id = ?", u.ID).Count(&count)
	if count != 0 {
		t.Fatal("revoked user's management token survived")
	}
	audit, err := s.UserRoleAudit()
	if err != nil || len(audit) != 2 || audit[0].BeforeRole != "operator" || audit[0].AfterRole != "user" {
		t.Fatalf("audit=%+v %v", audit, err)
	}
}
