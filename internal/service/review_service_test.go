package service

import (
	"math"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
)

func TestCalculateNewRating(t *testing.T) {
	tests := []struct {
		name         string
		currentAvg   float64
		currentCount int
		newScore     int
		wantAvg      float64
		wantCount    int
	}{
		{
			name:         "최초 1개 리뷰 등록 (5점)",
			currentAvg:   0,
			currentCount: 0,
			newScore:     5,
			wantAvg:      5.0,
			wantCount:    1,
		},
		{
			name:         "5점 1개에서 3점 추가 -> 평균 4.0",
			currentAvg:   5.0,
			currentCount: 1,
			newScore:     3,
			wantAvg:      4.0,
			wantCount:    2,
		},
		{
			name:         "4점 2개에서 1점 추가 -> 평균 3.0",
			currentAvg:   4.0,
			currentCount: 2,
			newScore:     1,
			wantAvg:      3.0,
			wantCount:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAvg, gotCount := CalculateNewRating(tt.currentAvg, tt.currentCount, tt.newScore)
			if math.Abs(gotAvg-tt.wantAvg) > 0.0001 {
				t.Errorf("CalculateNewRating() gotAvg = %v, want %v", gotAvg, tt.wantAvg)
			}
			if gotCount != tt.wantCount {
				t.Errorf("CalculateNewRating() gotCount = %v, want %v", gotCount, tt.wantCount)
			}
		})
	}
}

func TestDetermineAuthorName(t *testing.T) {
	tests := []struct {
		name       string
		authorType string
		customName string
		userName   string
		want       string
	}{
		{"익명", "anon", "무시됨", "홍길동", "익명"},
		{"실명", "real", "", "홍길동", "홍길동"},
		{"마스킹", "masked", "", "홍길동", "홍*동"},
		{"커스텀 닉네임", "custom", "성결대미식가", "홍길동", "성결대미식가"},
		{"커스텀 닉네임 10자 초과 자르기", "custom", "가나다라마바사아자차카타파하", "홍길동", "가나다라마바사아자차"},
		{"커스텀 닉네임 공백 입력 시 마스킹 대체", "custom", "   ", "홍길동", "홍*동"},
		{"기본값은 마스킹", "unknown", "", "김철수", "김*수"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineAuthorName(tt.authorType, tt.customName, tt.userName)
			if got != tt.want {
				t.Errorf("DetermineAuthorName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddReview_Transactional(t *testing.T) {
	// In-memory SQLite DB로 트랜잭션 테스트
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("in-memory db open failed: %v", err)
	}

	if err := db.AutoMigrate(&model.Restaurant{}, &model.Rating{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	restRepo := repository.NewRestaurantRepository(db)
	revRepo := repository.NewReviewRepository(db)
	svc := NewReviewService(db, restRepo, revRepo)

	// 초기 테스트 식당 생성
	testRes := model.Restaurant{
		Title:       "테스트식당",
		AvgRating:   4.0,
		RatingCount: 1,
	}
	db.Create(&testRes)

	// 1. 유효하지 않은 별점 (6점) 등록 시도 -> 에러 및 롤백 확인
	_, _, err = svc.AddReview(model.RateRequest{
		RestaurantID: testRes.ID,
		Score:        6,
	}, "홍길동")
	if err != ErrInvalidScore {
		t.Errorf("expected ErrInvalidScore, got %v", err)
	}

	// 2. 정상 별점 (2점) 등록 -> 트랜잭션 정상 커밋 및 평균 갱신 확인
	// (4.0 * 1 + 2) / 2 = 3.0
	rating, newAvg, err := svc.AddReview(model.RateRequest{
		RestaurantID: testRes.ID,
		Score:        2,
		Comment:      "맛있어요",
		AuthorType:   "anon",
	}, "홍길동")

	if err != nil {
		t.Fatalf("AddReview failed: %v", err)
	}

	if math.Abs(newAvg-3.0) > 0.0001 {
		t.Errorf("expected newAvg 3.0, got %v", newAvg)
	}

	if rating.AuthorName != "익명" {
		t.Errorf("expected AuthorName '익명', got %q", rating.AuthorName)
	}

	// DB 식당 데이터 갱신 여부 검증
	updatedRes, err := restRepo.FindByID(testRes.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if updatedRes.RatingCount != 2 {
		t.Errorf("expected RatingCount 2, got %d", updatedRes.RatingCount)
	}
	if math.Abs(updatedRes.AvgRating-3.0) > 0.0001 {
		t.Errorf("expected AvgRating 3.0, got %v", updatedRes.AvgRating)
	}

	// 3. GetReviews 호출 시 ReviewResponse DTO에 개인정보가 차단되고 AuthorName이 익명인지 검증
	reviews, err := svc.GetReviews(testRes.ID)
	if err != nil {
		t.Fatalf("GetReviews failed: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
	if reviews[0].AuthorName != "익명" {
		t.Errorf("expected AuthorName '익명', got %q", reviews[0].AuthorName)
	}
}
