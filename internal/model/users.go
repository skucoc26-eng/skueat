package model

import "time"

type User struct {
	ID          uint   `gorm:"primaryKey"`
	KakaoUserID int64  `json:"-" gorm:"not null;uniqueIndex"`
	Nickname    string `gorm:"not null"`
	Role        string `gorm:"not null;default:user"`
	CreatedAt   time.Time
	LastLoginAt time.Time
}

type UserRoleChange struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UserID     uint      `json:"user_id" gorm:"index"`
	BeforeRole string    `json:"before_role"`
	AfterRole  string    `json:"after_role"`
	Admin      string    `json:"admin"`
}
