package repository

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"restaurant-api/internal/model"
)

func TestInitDBSyncsCatalogWithoutResettingRatings(t *testing.T) {
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	writeCatalog := func(items []model.Restaurant) {
		t.Helper()
		data, err := json.Marshal(items)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("restaurants.json", data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	original := model.Restaurant{Title: "기존 식당", Addr: "기존 주소", Food: "한식", X: 126.93, Y: 37.38, URL: "http://place.map.kakao.com/100"}
	writeCatalog([]model.Restaurant{original})
	dbPath := filepath.Join(dir, "catalog.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var existing model.Restaurant
	if err := db.Where("url = ?", original.URL).First(&existing).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&existing).Updates(map[string]any{"avg_rating": 4.0, "rating_count": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Rating{RestaurantID: existing.ID, UserID: "legacy", Score: 4}).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	updated := original
	updated.Title = "새 이름"
	updated.Addr = "새 주소"
	updated.URL = "https://place.map.kakao.com/100"
	added := model.Restaurant{Title: "새 식당", Addr: "추가 주소", Food: "카페", X: 126.94, Y: 37.39, URL: "https://place.map.kakao.com/200"}
	writeCatalog([]model.Restaurant{updated, added})
	db, err = InitDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	reopenedSQLDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedSQLDB.Close()
	var restaurants []model.Restaurant
	if err := db.Order("id").Find(&restaurants).Error; err != nil {
		t.Fatal(err)
	}
	if len(restaurants) != 2 {
		t.Fatalf("restaurants=%d, want 2", len(restaurants))
	}
	if restaurants[0].ID != existing.ID || restaurants[0].Title != "새 이름" || restaurants[0].AvgRating != 4 || restaurants[0].RatingCount != 1 {
		t.Fatalf("existing restaurant lost identity, details, or ratings: %+v", restaurants[0])
	}
	var ratings int64
	if err := db.Model(&model.Rating{}).Where("restaurant_id = ?", existing.ID).Count(&ratings).Error; err != nil {
		t.Fatal(err)
	}
	if ratings != 1 {
		t.Fatalf("ratings=%d, want 1", ratings)
	}
	if err := syncRestaurantCatalog(db); err != nil {
		t.Fatal(err)
	}
	var afterRepeat int64
	if err := db.Model(&model.Restaurant{}).Count(&afterRepeat).Error; err != nil {
		t.Fatal(err)
	}
	if afterRepeat != 2 {
		t.Fatalf("repeat sync produced %d restaurants, want 2", afterRepeat)
	}
}

type legacyRating struct {
	gorm.Model
	RestaurantID uint
	UserID       string
	AuthorName   string
	Score        int
	Comment      string
}

func (legacyRating) TableName() string { return "ratings" }

func TestRatingMigrationKeepsLegacyDuplicatesAndProtectsKakaoIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&legacyRating{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := db.Create(&legacyRating{RestaurantID: 1, UserID: "same nickname", Score: 5}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&model.Rating{}); err != nil {
		t.Fatalf("legacy migration failed: %v", err)
	}
	var count int64
	if err := db.Model(&model.Rating{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("legacy ratings=%d, want 2", count)
	}
	userID := int64(123)
	if err := db.Create(&model.Rating{RestaurantID: 1, KakaoUserID: &userID, Score: 5}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Rating{RestaurantID: 1, KakaoUserID: &userID, Score: 5}).Error; err == nil {
		t.Fatal("duplicate Kakao account rating bypassed unique index")
	}
	if err := NewReviewRepository(db).CreateWithTx(nil, &model.Rating{RestaurantID: 1, KakaoUserID: &userID, Score: 5}); !errors.Is(err, ErrRatingConflict) {
		t.Fatalf("duplicate repository insert error=%v, want ErrRatingConflict", err)
	}
}
