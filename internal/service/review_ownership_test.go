package service

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
)

func TestLegacyReviewOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reviews.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}); err != nil {
		t.Fatal(err)
	}
	rest := model.Restaurant{Title: "테스트"}
	if err := db.Create(&rest).Error; err != nil {
		t.Fatal(err)
	}
	// Even a legacy nickname identical to the new account ID must not confer ownership.
	legacy := model.Rating{RestaurantID: rest.ID, UserID: "kakao_123", AuthorName: "익명", Score: 2, Comment: "기존 리뷰"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	// Exercise migration from a schema without the new ownership column.
	if err := db.Migrator().DropColumn(&model.Rating{}, "AccountID"); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Rating{}); err != nil {
		t.Fatal(err)
	}
	svc := NewReviewService(db, repository.NewRestaurantRepository(db), repository.NewReviewRepository(db))
	req := model.RateRequest{RestaurantID: rest.ID, Score: 4, Comment: "새 리뷰"}
	if _, _, _, err := svc.AddReview(req, "", "kakao_123"); err != ErrUnauthenticated {
		t.Fatalf("missing account: %v", err)
	}
	created, avg, updated, err := svc.AddReview(req, "kakao_123", "kakao_123")
	if err != nil || updated || avg != 3 {
		t.Fatalf("create: avg=%v updated=%v err=%v", avg, updated, err)
	}
	if created.ID == legacy.ID {
		t.Fatal("claimed legacy review")
	}
	req.Score = 5
	modified, _, updated, err := svc.AddReview(req, "kakao_123", "변경된 닉네임")
	if err != nil || !updated || modified.ID != created.ID {
		t.Fatalf("update: %v", err)
	}
	if err := db.First(&legacy, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.Score != 2 || legacy.Comment != "기존 리뷰" || legacy.AccountID != "" {
		t.Fatalf("legacy changed: %+v", legacy)
	}
	// Another account with the same nickname owns a separate review.
	if _, _, updated, err := svc.AddReview(req, "kakao_456", "kakao_123"); err != nil || updated {
		t.Fatalf("same nickname: %v", err)
	}
	var count int64
	if err := db.Model(&model.Rating{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
