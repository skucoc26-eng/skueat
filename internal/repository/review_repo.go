package repository

import (
	"gorm.io/gorm"

	"restaurant-api/internal/model"
)

// ReviewRepository 리뷰 및 별점 데이터 접근 인터페이스
type ReviewRepository interface {
	FindByRestaurantID(restaurantID uint) ([]model.Rating, error)
	FindByRestaurantAndUser(tx *gorm.DB, restaurantID uint, userID string) (*model.Rating, error)
	CreateWithTx(tx *gorm.DB, rating *model.Rating) error
	UpdateWithTx(tx *gorm.DB, rating *model.Rating) error
}

type reviewRepository struct {
	db *gorm.DB
}

// NewReviewRepository ReviewRepository 구현체 생성
func NewReviewRepository(db *gorm.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) FindByRestaurantID(restaurantID uint) ([]model.Rating, error) {
	var reviews []model.Rating
	if err := r.db.Where("restaurant_id = ?", restaurantID).Order("id desc").Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) FindByRestaurantAndUser(tx *gorm.DB, restaurantID uint, userID string) (*model.Rating, error) {
	db := r.db
	if tx != nil {
		db = tx
	}
	var review model.Rating
	if err := db.Where("restaurant_id = ? AND account_id = ?", restaurantID, userID).First(&review).Error; err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) CreateWithTx(tx *gorm.DB, rating *model.Rating) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Create(rating).Error
}

func (r *reviewRepository) UpdateWithTx(tx *gorm.DB, rating *model.Rating) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Save(rating).Error
}
