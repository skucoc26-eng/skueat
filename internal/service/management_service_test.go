package service

import (
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
)

func managementDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}, &model.UsageEvent{}, &model.ReviewModeration{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestModerationPreservesReviewAndRecalculatesPublicScore(t *testing.T) {
	db := managementDB(t)
	r := model.Restaurant{Title: "후로", AvgRating: 3, RatingCount: 2}
	db.Create(&r)
	u1, u2 := int64(10), int64(20)
	a := model.Rating{RestaurantID: r.ID, Score: 1, Comment: "review", KakaoUserID: &u1}
	b := model.Rating{RestaurantID: r.ID, Score: 5, KakaoUserID: &u2}
	db.Create(&a)
	db.Create(&b)
	s := NewManagementService(db)
	if err := s.Moderate(a.ID, true, "스팸", "admin"); err != nil {
		t.Fatal(err)
	}
	public, err := repository.NewReviewRepository(db).FindByRestaurantID(r.ID)
	if err != nil || len(public) != 1 || public[0].Score != 5 {
		t.Fatalf("public=%+v err=%v", public, err)
	}
	db.First(&r, r.ID)
	if r.RatingCount != 1 || r.AvgRating != 5 {
		t.Fatalf("stats=%+v", r)
	}
	var stored model.Rating
	db.First(&stored, a.ID)
	if !stored.Hidden || stored.Comment != "review" {
		t.Fatal("original lost")
	}
	var count int64
	db.Model(&model.Rating{}).Where("restaurant_id = ? AND kakao_user_id = ?", r.ID, u1).Count(&count)
	if count != 1 {
		t.Fatal("duplicate identity lost")
	}
	if err := s.Moderate(a.ID, false, "오판 복구", "admin"); err != nil {
		t.Fatal(err)
	}
	db.First(&r, r.ID)
	if r.RatingCount != 2 || r.AvgRating != 3 {
		t.Fatalf("restored stats=%+v", r)
	}
	db.Model(&model.ReviewModeration{}).Count(&count)
	if count != 2 {
		t.Fatalf("audit count=%d", count)
	}
	if err := s.Moderate(a.ID, true, "", "admin"); err == nil {
		t.Fatal("missing reason accepted")
	}
}

func TestUsageStatisticsBoundariesAndUniqueVisitors(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	now := time.Date(2026, 10, 29, 1, 0, 0, 0, time.FixedZone("KST", 9*3600))
	r := model.Restaurant{Title: "후로"}
	db.Create(&r)
	for _, v := range []string{"a", "a", "b"} {
		if err := s.RecordEvent(model.UsageEvent{Visitor: v, Type: "page_view"}, now); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "search", Query: "후로"}, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "detail", RestaurantID: r.ID}, now); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.UsageEvent{Visitor: "old", Type: "page_view", CreatedAt: now.AddDate(0, 0, -100)})
	stats, err := s.Statistics(1, now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.PageViews != 3 || stats.Visitors != 2 || stats.Searches != 2 || stats.Details != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if len(stats.Daily) != 1 || stats.Daily[0].Day != "2026-10-29" {
		t.Fatalf("daily=%+v", stats.Daily)
	}
	if len(stats.Queries) != 1 || stats.Queries[0].Count != 2 {
		t.Fatalf("queries=%+v", stats.Queries)
	}
	if len(stats.Restaurants) != 1 || stats.Restaurants[0].Title != "후로" {
		t.Fatal("missing restaurant ranking")
	}
	if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "unknown"}, now); err == nil {
		t.Fatal("unknown event accepted")
	}
	if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "detail", RestaurantID: 999}, now); err == nil {
		t.Fatal("unknown restaurant accepted")
	}
	// 15:00 UTC is midnight KST; the previous day must not enter today's count.
	db.Create(&model.UsageEvent{Visitor: "yesterday", Type: "page_view", CreatedAt: time.Date(2026, 10, 28, 14, 59, 59, 0, time.UTC)})
	stats, err = s.Statistics(1, now)
	if err != nil || stats.PageViews != 3 {
		t.Fatalf("KST boundary stats=%+v err=%v", stats, err)
	}
	var old int64
	db.Model(&model.UsageEvent{}).Where("visitor = ?", "old").Count(&old)
	if old != 0 {
		t.Fatal("expired analytics retained")
	}
}

func TestModerationRollsBackWhenAuditFails(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	r := model.Restaurant{Title: "test", AvgRating: 1, RatingCount: 1}
	db.Create(&r)
	review := model.Rating{RestaurantID: r.ID, Score: 1}
	db.Create(&review)
	if err := db.Exec("CREATE TRIGGER fail_audit BEFORE INSERT ON review_moderations BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.Moderate(review.ID, true, "spam", "admin"); err == nil {
		t.Fatal("audit failure ignored")
	}
	db.First(&review, review.ID)
	db.First(&r, r.ID)
	if review.Hidden || r.RatingCount != 1 || r.AvgRating != 1 {
		t.Fatal("partial moderation committed")
	}
}

func TestUsageRateLimit(t *testing.T) {
	db := managementDB(t)
	s := NewManagementService(db)
	now := time.Now()
	for i := 0; i < 120; i++ {
		if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "page_view"}, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordEvent(model.UsageEvent{Visitor: "a", Type: "page_view"}, now); !errors.Is(err, ErrUsageLimit) {
		t.Fatalf("rate limit=%v", err)
	}
}
