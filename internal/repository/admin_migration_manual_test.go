//go:build migrationcheck

package repository

import (
	"os"
	"reflect"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"restaurant-api/internal/model"
)

// Run only on a disposable copy: SKUEAT_MIGRATION_DB=... go test -tags=migrationcheck ./internal/repository -run TestAdminMigrationSnapshot
func TestAdminMigrationSnapshot(t *testing.T) {
	path := os.Getenv("SKUEAT_MIGRATION_DB")
	if path == "" {
		t.Skip("requires a disposable snapshot copy")
	}
	before, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var restaurantsBefore, ratingsBefore []map[string]interface{}
	if err := before.Table("restaurants").Order("id").Find(&restaurantsBefore).Error; err != nil {
		t.Fatal(err)
	}
	if err := before.Table("ratings").Order("id").Find(&ratingsBefore).Error; err != nil {
		t.Fatal(err)
	}
	sqlBefore, _ := before.DB()
	if err := sqlBefore.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlAfter, _ := after.DB()
	defer sqlAfter.Close()
	var restaurantsAfter, ratingsAfter []map[string]interface{}
	if err := after.Table("restaurants").Order("id").Find(&restaurantsAfter).Error; err != nil {
		t.Fatal(err)
	}
	if err := after.Table("ratings").Order("id").Find(&ratingsAfter).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restaurantsBefore, restaurantsAfter) {
		t.Fatal("restaurant records changed during migration")
	}
	if len(ratingsBefore) != len(ratingsAfter) {
		t.Fatal("rating count changed during migration")
	}
	for i, row := range ratingsBefore {
		for key, value := range row {
			if !reflect.DeepEqual(value, ratingsAfter[i][key]) {
				t.Fatalf("rating %d field %s changed", i, key)
			}
		}
	}
	for _, table := range []interface{}{&model.AdminSession{}, &model.UsageEvent{}, &model.ReviewModeration{}} {
		if !after.Migrator().HasTable(table) {
			t.Fatal("management table missing")
		}
	}
	var hidden int64
	if err := after.Model(&model.Rating{}).Where("hidden = ?", true).Count(&hidden).Error; err != nil || hidden != 0 {
		t.Fatal("legacy ratings incorrectly hidden")
	}
	t.Logf("preserved %d restaurants and %d ratings; management tables ready", len(restaurantsAfter), len(ratingsAfter))
}
