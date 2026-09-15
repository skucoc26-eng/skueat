package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
	"restaurant-api/internal/utils"
)

var (
	ErrRestaurantNotFound = errors.New("식당을 찾을 수 없습니다")
	ErrInvalidScore       = errors.New("별점은 1점에서 5점 사이여야 합니다")
)

// ReviewService 리뷰 및 평점 비즈니스 로직 인터페이스
type ReviewService interface {
	AddReview(req model.RateRequest, userName string) (*model.Rating, float64, error)
	GetReviews(restaurantID uint) ([]model.ReviewResponse, error)
}

type reviewService struct {
	db             *gorm.DB
	restaurantRepo repository.RestaurantRepository
	reviewRepo     repository.ReviewRepository
}

// NewReviewService ReviewService 구현체 생성
func NewReviewService(db *gorm.DB, restRepo repository.RestaurantRepository, revRepo repository.ReviewRepository) ReviewService {
	return &reviewService{
		db:             db,
		restaurantRepo: restRepo,
		reviewRepo:     revRepo,
	}
}

// AddReview 트랜잭션을 통해 리뷰 등록 및 식당 통계(평균 평점, 참여 수)를 원자적으로 갱신
func (s *reviewService) AddReview(req model.RateRequest, userName string) (*model.Rating, float64, error) {
	if req.Score < 1 || req.Score > 5 {
		return nil, 0, ErrInvalidScore
	}

	authorName := DetermineAuthorName(req.AuthorType, req.CustomName, userName)

	var rating model.Rating
	var newAvg float64

	// 원자적 처리를 위한 DB 트랜잭션
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 식당 존재 여부 및 현재 통계 조회
		res, err := s.restaurantRepo.FindByIDWithTx(tx, req.RestaurantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRestaurantNotFound
			}
			return err
		}

		// 2. 신규 평균 평점 및 개수 산출
		var newCount int
		newAvg, newCount = CalculateNewRating(res.AvgRating, res.RatingCount, req.Score)

		// 3. 리뷰 레코드 생성
		rating = model.Rating{
			RestaurantID: req.RestaurantID,
			UserID:       userName,
			AuthorName:   authorName,
			Score:        req.Score,
			Comment:      req.Comment,
		}
		if err := s.reviewRepo.CreateWithTx(tx, &rating); err != nil {
			return err
		}

		// 4. 식당 통계 업데이트
		if err := s.restaurantRepo.UpdateStatsWithTx(tx, req.RestaurantID, newAvg, newCount); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return &rating, newAvg, nil
}

// GetReviews 식당의 리뷰 목록 조회 (개인정보를 완전히 차단한 ReviewResponse DTO 반환)
func (s *reviewService) GetReviews(restaurantID uint) ([]model.ReviewResponse, error) {
	reviews, err := s.reviewRepo.FindByRestaurantID(restaurantID)
	if err != nil {
		return nil, err
	}

	resList := make([]model.ReviewResponse, len(reviews))
	for i, r := range reviews {
		author := r.AuthorName
		if author == "" {
			author = utils.MaskName(r.UserID)
		}
		resList[i] = model.ReviewResponse{
			ID:           r.ID,
			RestaurantID: r.RestaurantID,
			AuthorName:   author,
			Score:        r.Score,
			Comment:      r.Comment,
			CreatedAt:    r.CreatedAt,
		}
	}

	return resList, nil
}

// DetermineAuthorName 옵션에 따른 표시 닉네임 결정
func DetermineAuthorName(authorType, customName, userName string) string {
	trimmedCustom := strings.TrimSpace(customName)

	switch authorType {
	case "anon":
		return "익명"
	case "custom":
		if trimmedCustom != "" {
			runes := []rune(trimmedCustom)
			if len(runes) > 10 {
				return string(runes[:10])
			}
			return trimmedCustom
		}
		return utils.MaskName(userName)
	case "real":
		return userName
	case "masked":
		fallthrough
	default:
		return utils.MaskName(userName)
	}
}

// CalculateNewRating 기존 평균과 신규 점수로 새 평균 평점 및 총 개수 계산
func CalculateNewRating(currentAvg float64, currentCount, newScore int) (float64, int) {
	newCount := currentCount + 1
	newAvg := (currentAvg*float64(currentCount) + float64(newScore)) / float64(newCount)
	return newAvg, newCount
}
