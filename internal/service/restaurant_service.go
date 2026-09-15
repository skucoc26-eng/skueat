package service

import (
	"restaurant-api/internal/model"
	"restaurant-api/internal/repository"
)

// RestaurantService 맛집 관련 비즈니스 로직 인터페이스
type RestaurantService interface {
	GetRestaurants(category, search string) ([]model.Restaurant, error)
	GetRandomRestaurant() (*model.Restaurant, error)
}

type restaurantService struct {
	repo repository.RestaurantRepository
}

// NewRestaurantService RestaurantService 구현체 생성
func NewRestaurantService(repo repository.RestaurantRepository) RestaurantService {
	return &restaurantService{repo: repo}
}

func (s *restaurantService) GetRestaurants(category, search string) ([]model.Restaurant, error) {
	return s.repo.FindAll(category, search)
}

func (s *restaurantService) GetRandomRestaurant() (*model.Restaurant, error) {
	return s.repo.FindRandom()
}
