package service

import (
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"restaurant-api/internal/config"
	"restaurant-api/internal/model"
)

func (s *ManagementService) LoginUser(kakaoID int64, nickname, token, oldToken string, now time.Time) (model.User, error) {
	user := model.User{}
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = "카카오 사용자"
	}
	if _, err := hex.DecodeString(token); err != nil || len(token) != 64 || kakaoID <= 0 || !utf8.ValidString(nickname) || utf8.RuneCountInString(nickname) > 80 {
		return user, ErrManagementInput
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		user = model.User{KakaoUserID: kakaoID, Nickname: nickname, Role: "user", CreatedAt: now.UTC(), LastLoginAt: now.UTC()}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "kakao_user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"nickname", "last_login_at"})}).Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Where("kakao_user_id = ?", kakaoID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at <= ? OR (token = ? AND user_id <> 0)", now.UTC(), oldToken).Delete(&model.AdminSession{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.AdminSession{Token: token, UserID: user.ID, ExpiresAt: now.Add(time.Duration(config.SessionDuration) * time.Second).UTC()}).Error
	})
	return user, err
}

type ManagedUser struct {
	ID           uint      `json:"id"`
	Nickname     string    `json:"nickname"`
	Role         string    `json:"role"`
	FirstLoginAt time.Time `json:"first_login_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
	ReviewCount  int64     `json:"review_count"`
}

func (s *ManagementService) Users(search string, page int) ([]ManagedUser, int64, error) {
	search = strings.TrimSpace(search)
	if page < 1 || page > 10000 || utf8.RuneCountInString(search) > 80 {
		return nil, 0, ErrManagementInput
	}
	q := s.db.Model(&model.User{})
	if search != "" {
		q = q.Where("users.nickname LIKE ? OR CAST(users.id AS TEXT) = ?", "%"+search+"%", search)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := []ManagedUser{}
	err := q.Select("users.id, users.nickname, users.role, users.created_at AS first_login_at, users.last_login_at, COUNT(ratings.id) AS review_count").
		Joins("LEFT JOIN ratings ON ratings.kakao_user_id = users.kakao_user_id AND ratings.deleted_at IS NULL").
		Group("users.id").Order("users.id DESC").Offset((page - 1) * 50).Limit(50).Scan(&rows).Error
	return rows, total, err
}

func (s *ManagementService) SetUserRole(id uint, role, admin string) error {
	if id == 0 || (role != "user" && role != "operator") || admin == "" {
		return ErrManagementInput
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}
		if user.Role == role {
			return nil
		}
		before := user.Role
		if err := tx.Model(&user).Update("role", role).Error; err != nil {
			return err
		}
		if role == "user" {
			if err := tx.Where("user_id = ?", id).Delete(&model.AdminSession{}).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.UserRoleChange{UserID: id, BeforeRole: before, AfterRole: role, Admin: admin}).Error
	})
}

func (s *ManagementService) UserRoleAudit() ([]model.UserRoleChange, error) {
	rows := []model.UserRoleChange{}
	err := s.db.Order("id DESC").Limit(50).Find(&rows).Error
	return rows, err
}
