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
	ErrUnauthenticated    = errors.New("다시 로그인해 주세요")
)

// ReviewService 리뷰 및 평점 비즈니스 로직 인터페이스
type ReviewService interface {
	AddReview(req model.RateRequest, userID, userName string) (*model.Rating, float64, bool, error)
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

// AddReview 트랜잭션을 통해 1인 1리뷰(Upsert: 기존 리뷰 수정 또는 신규 생성) 및 식당 통계를 원자적으로 재계산
func (s *reviewService) AddReview(req model.RateRequest, userID, userName string) (*model.Rating, float64, bool, error) {
	if req.Score < 1 || req.Score > 5 {
		return nil, 0, false, ErrInvalidScore
	}

	if userID == "" {
		return nil, 0, false, ErrUnauthenticated
	}

	authorName := DetermineAuthorName(req.AuthorType, req.CustomName, userName)

	var rating model.Rating
	var newAvg float64
	var isUpdated bool

	// 원자적 처리를 위한 DB 트랜잭션
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 식당 존재 여부 확인
		_, err := s.restaurantRepo.FindByIDWithTx(tx, req.RestaurantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRestaurantNotFound
			}
			return err
		}

		// 2. 해당 사용자의 기존 리뷰 여부 확인 (1인 1리뷰 정책)
		existing, err := s.reviewRepo.FindByRestaurantAndUser(tx, req.RestaurantID, userID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if existing != nil {
			// 기존 리뷰가 존재하면 수정 (Update)
			existing.AuthorName = authorName
			existing.Score = req.Score
			existing.Comment = req.Comment
			if err := s.reviewRepo.UpdateWithTx(tx, existing); err != nil {
				return err
			}
			rating = *existing
			isUpdated = true
		} else {
			// 기존 리뷰가 없으면 신규 등록 (Create)
			rating = model.Rating{
				RestaurantID: req.RestaurantID,
				UserID:       userID,
				AccountID:    userID,
				AuthorName:   authorName,
				Score:        req.Score,
				Comment:      req.Comment,
			}
			if err := s.reviewRepo.CreateWithTx(tx, &rating); err != nil {
				return err
			}
			isUpdated = false
		}

		// 3. 식당 통계(평균 평점 및 리뷰 개수)를 DB 레벨에서 원자적으로 재집계 및 갱신 (Lost Update 방지)
		calcAvg, _, err := s.restaurantRepo.RecalculateStatsWithTx(tx, req.RestaurantID)
		if err != nil {
			return err
		}
		newAvg = calcAvg

		return nil
	})

	if err != nil {
		return nil, 0, false, err
	}

	return &rating, newAvg, isUpdated, nil
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
