package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"restaurant-api/internal/model"
)

func TestFindAllSearchesFood(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Restaurant{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Restaurant{Title: "테스트 식당", Addr: "안양시", Food: "양식"}).Error; err != nil {
		t.Fatal(err)
	}
	got, err := NewRestaurantRepository(db).FindAll("", "양식")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Food != "양식" {
		t.Fatalf("food-only search returned %+v, want one 양식 restaurant", got)
	}
}
