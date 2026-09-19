package model

import (
	"time"

	"gorm.io/gorm"
)

// Restaurant 식당 엔티티
type Restaurant struct {
	gorm.Model
	Title       string  `json:"title"`
	Addr        string  `json:"addr"`
	Food        string  `json:"food"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	URL         string  `json:"url"`
	AvgRating   float64 `json:"avg_rating" gorm:"column:avg_rating;default:0"`     // 평균 별점
	RatingCount int     `json:"rating_count" gorm:"column:rating_count;default:0"` // 참여 인원
}

// Rating 별점 및 리뷰 기록 엔티티
type Rating struct {
	gorm.Model
	RestaurantID uint `json:"restaurant_id" gorm:"index"`
	// AccountID는 인증된 계정의 소유권입니다. 기존 UserID(닉네임)에서 자동 변환하지 않습니다.
	AccountID  string `json:"-" gorm:"index"`
	UserID     string `json:"-" gorm:"index"` // 내부 식별용 (API 응답에서 절대 노출되지 않음)
	AuthorName string `json:"author_name"`    // 화면에 표시될 닉네임 (마스킹, 익명, 실명 등)
	Score      int    `json:"score"`
	Comment    string `json:"comment" gorm:"type:text"` // 한 줄 평
}

// ReviewResponse 클라이언트 공개용 리뷰 응답 DTO (사용자 실제 식별자 제외)
type ReviewResponse struct {
	ID           uint      `json:"id"`
	RestaurantID uint      `json:"restaurant_id"`
	AuthorName   string    `json:"author_name"`
	Score        int       `json:"score"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// RateRequest 리뷰/별점 등록 요청 폼 바인딩 구조체
type RateRequest struct {
	RestaurantID uint   `form:"restaurant_id" binding:"required,gt=0"`
	Score        int    `form:"score" binding:"required,min=1,max=5"`
	Comment      string `form:"comment" binding:"max=300"`
	AuthorType   string `form:"author_type"`
	CustomName   string `form:"custom_name"`
}

// KakaoTokenResponse 카카오 OAuth 토큰 발급 응답 구조체
type KakaoTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// KakaoUserResponse 카카오 OAuth 사용자 정보 응답 구조체
type KakaoUserResponse struct {
	ID         int64 `json:"id"`
	Properties struct {
		Nickname string `json:"nickname"`
	} `json:"properties"`
	KakaoAccount struct {
		Profile struct {
			Nickname string `json:"nickname"`
		} `json:"profile"`
	} `json:"kakao_account"`
}
