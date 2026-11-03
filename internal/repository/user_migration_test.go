package repository

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"restaurant-api/internal/model"
)

func TestUserMigrationPreservesOwnerSessionsAndExistingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}); err != nil {
		t.Fatal(err)
	}
	restaurant := model.Restaurant{Title: "기존 식당", RatingCount: 1, AvgRating: 3}
	if err := db.Create(&restaurant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Rating{RestaurantID: restaurant.ID, Score: 3, Comment: "기존 리뷰"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE admin_sessions (token TEXT PRIMARY KEY, expires_at DATETIME)").Error; err != nil {
		t.Fatal(err)
	}
	expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := db.Exec("INSERT INTO admin_sessions (token,expires_at) VALUES (?,?)", "existing-owner", expiry).Error; err != nil {
		t.Fatal(err)
	}
	old, _ := db.DB()
	old.Close()
	db, err = InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	var session model.AdminSession
	if err := db.First(&session, "token = ?", "existing-owner").Error; err != nil || session.UserID != 0 || !session.ExpiresAt.Equal(expiry) {
		t.Fatalf("owner session changed: %+v %v", session, err)
	}
	var rating model.Rating
	if err := db.First(&rating).Error; err != nil || rating.Comment != "기존 리뷰" || rating.Score != 3 {
		t.Fatal("legacy review changed")
	}
	var current model.Restaurant
	if err := db.First(&current).Error; err != nil || current.ID != restaurant.ID || current.RatingCount != 1 || current.AvgRating != 3 {
		t.Fatal("legacy restaurant changed")
	}
	for _, table := range []interface{}{&model.User{}, &model.UserRoleChange{}} {
		if !db.Migrator().HasTable(table) {
			t.Fatal("user table missing")
		}
	}
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count != 0 {
		t.Fatal("legacy login dates were invented")
	}
}
