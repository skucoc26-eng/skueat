package model

import "time"

type AdminSession struct {
	Token     string    `gorm:"primaryKey"`
	ExpiresAt time.Time `gorm:"index"`
}

type ReviewModeration struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ReviewID  uint      `json:"review_id" gorm:"index"`
	Admin     string    `json:"admin"`
	Hidden    bool      `json:"hidden"`
	Reason    string    `json:"reason"`
}

type UsageEvent struct {
	ID           uint      `json:"-"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	Visitor      string    `json:"-" gorm:"index"`
	Type         string    `json:"type" gorm:"index"`
	RestaurantID uint      `json:"restaurant_id" gorm:"index"`
	Query        string    `json:"query"`
	Category     string    `json:"category"`
	Source       string    `json:"source"`
}
