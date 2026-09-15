package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"restaurant-api/internal/service"
)

// RestaurantHandler 맛집 관련 HTTP 핸들러
type RestaurantHandler struct {
	svc service.RestaurantService
}

// NewRestaurantHandler RestaurantHandler 생성
func NewRestaurantHandler(svc service.RestaurantService) *RestaurantHandler {
	return &RestaurantHandler{svc: svc}
}

// GetRestaurants 맛집 목록 조회 API (GET /api/restaurants)
func (h *RestaurantHandler) GetRestaurants(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")

	list, err := h.svc.GetRestaurants(category, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "식당 목록을 불러오는 중 오류가 발생했습니다."})
		return
	}

	c.JSON(http.StatusOK, list)
}

// GetRandomRestaurant 무작위 맛집 1곳 추천 API (GET /api/restaurants/random)
func (h *RestaurantHandler) GetRandomRestaurant(c *gin.Context) {
	pick, err := h.svc.GetRandomRestaurant()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "데이터를 찾을 수 없습니다."})
		return
	}

	c.JSON(http.StatusOK, pick)
}
