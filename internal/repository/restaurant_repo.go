package repository

import (
	"gorm.io/gorm"

	"restaurant-api/internal/model"
)

// RestaurantRepository 맛집 데이터 접근 인터페이스
type RestaurantRepository interface {
	FindAll(category, search string) ([]model.Restaurant, error)
	FindByID(id uint) (*model.Restaurant, error)
	FindByIDWithTx(tx *gorm.DB, id uint) (*model.Restaurant, error)
	FindRandom() (*model.Restaurant, error)
	UpdateStatsWithTx(tx *gorm.DB, id uint, avgRating float64, count int) error
	RecalculateStatsWithTx(tx *gorm.DB, id uint) (float64, int, error)
}

type restaurantRepository struct {
	db *gorm.DB
}

// NewRestaurantRepository RestaurantRepository 구현체 생성
func NewRestaurantRepository(db *gorm.DB) RestaurantRepository {
	return &restaurantRepository{db: db}
}

func (r *restaurantRepository) FindAll(category, search string) ([]model.Restaurant, error) {
	var list []model.Restaurant
	query := r.db.Model(&model.Restaurant{})

	if category != "" && category != "all" {
		query = query.Where("food LIKE ?", "%"+category+"%")
	}
	if search != "" {
		query = query.Where("title LIKE ? OR addr LIKE ? OR food LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *restaurantRepository) FindByID(id uint) (*model.Restaurant, error) {
	var res model.Restaurant
	if err := r.db.First(&res, id).Error; err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *restaurantRepository) FindByIDWithTx(tx *gorm.DB, id uint) (*model.Restaurant, error) {
	var res model.Restaurant
	db := r.db
	if tx != nil {
		db = tx
	}
	if err := db.First(&res, id).Error; err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *restaurantRepository) FindRandom() (*model.Restaurant, error) {
	var pick model.Restaurant
	if err := r.db.Order("RANDOM()").First(&pick).Error; err != nil {
		return nil, err
	}
	return &pick, nil
}

func (r *restaurantRepository) UpdateStatsWithTx(tx *gorm.DB, id uint, avgRating float64, count int) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Model(&model.Restaurant{}).Where("id = ?", id).Updates(map[string]interface{}{
		"avg_rating":   avgRating,
		"rating_count": count,
	}).Error
}

func (r *restaurantRepository) RecalculateStatsWithTx(tx *gorm.DB, id uint) (float64, int, error) {
	db := r.db
	if tx != nil {
		db = tx
	}

	var stats struct {
		Avg   float64
		Count int
	}

	if err := db.Model(&model.Rating{}).
		Select("COALESCE(ROUND(AVG(score), 1), 0) as avg, COUNT(*) as count").
		Where("restaurant_id = ? AND deleted_at IS NULL", id).
		Scan(&stats).Error; err != nil {
		return 0, 0, err
	}

	if err := db.Model(&model.Restaurant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"avg_rating":   stats.Avg,
			"rating_count": stats.Count,
		}).Error; err != nil {
		return 0, 0, err
	}

	return stats.Avg, stats.Count, nil
}
