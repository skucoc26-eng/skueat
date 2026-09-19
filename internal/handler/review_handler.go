package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"restaurant-api/internal/model"
	"restaurant-api/internal/service"
)

// ReviewHandler 리뷰 및 평점 관련 HTTP 핸들러
type ReviewHandler struct {
	svc service.ReviewService
}

// NewReviewHandler ReviewHandler 생성
func NewReviewHandler(svc service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// AddReview 별점 평가 및 한 줄 평 등록 API (POST /api/rate)
func (h *ReviewHandler) AddReview(c *gin.Context) {
	session := sessions.Default(c)
	userID, userName, ok := getValidUser(session)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "로그인이 필요하거나 세션이 만료되었습니다."})
		return
	}

	var req model.RateRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "입력 형식이 올바르지 않습니다. (별점 1~5점 필수)"})
		return
	}

	_, newAvg, isUpdated, err := h.svc.AddReview(req, userID, userName)
	if err != nil {
		if errors.Is(err, service.ErrUnauthenticated) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrRestaurantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "식당을 찾을 수 없습니다."})
			return
		}
		if errors.Is(err, service.ErrInvalidScore) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "별점은 1점에서 5점 사이여야 합니다."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "평가를 저장하는 도중 오류가 발생했습니다."})
		return
	}

	msg := "평가가 완료되었습니다."
	if isUpdated {
		msg = "기존 평가가 성공적으로 수정되었습니다."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    msg,
		"new_avg":    newAvg,
		"is_updated": isUpdated,
	})
}

// GetReviews 특정 식당의 리뷰 목록 조회 API (GET /api/reviews)
func (h *ReviewHandler) GetReviews(c *gin.Context) {
	resIDStr := c.Query("restaurant_id")
	if resIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "restaurant_id가 필요합니다."})
		return
	}

	resID, err := strconv.ParseUint(resIDStr, 10, 32)
	if err != nil || resID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "유효하지 않은 restaurant_id입니다."})
		return
	}

	reviews, err := h.svc.GetReviews(uint(resID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "리뷰를 불러올 수 없습니다."})
		return
	}

	c.JSON(http.StatusOK, reviews)
}
