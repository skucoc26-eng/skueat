package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"restaurant-api/internal/model"
)

var ErrRatingConflict = errors.New("이미 등록된 사용자 평점입니다")

// ReviewRepository 리뷰 및 별점 데이터 접근 인터페이스
type ReviewRepository interface {
	FindByRestaurantID(restaurantID uint) ([]model.Rating, error)
	CreateWithTx(tx *gorm.DB, rating *model.Rating) error
	ExistsByRestaurantAndKakaoUserIDWithTx(tx *gorm.DB, restaurantID uint, userID int64) (bool, error)
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
	if err := r.db.Where("restaurant_id = ? AND hidden = ?", restaurantID, false).Order("id desc").Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) ExistsByRestaurantAndKakaoUserIDWithTx(tx *gorm.DB, restaurantID uint, userID int64) (bool, error) {
	var count int64
	err := tx.Model(&model.Rating{}).Where("restaurant_id = ? AND kakao_user_id = ?", restaurantID, userID).Count(&count).Error
	return count > 0, err
}

func (r *reviewRepository) CreateWithTx(tx *gorm.DB, rating *model.Rating) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "restaurant_id"}, {Name: "kakao_user_id"}},
		DoNothing: true,
	}).Create(rating)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRatingConflict
	}
	return nil
}
