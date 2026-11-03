package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"restaurant-api/internal/model"
	"restaurant-api/internal/service"
)

func (h *AdminHandler) access(c *gin.Context) (string, bool) {
	if h.authenticated(c) {
		return h.username(), true
	}
	_, kakaoID, valid := getValidUser(sessions.Default(c))
	token, _ := sessions.Default(c).Get("userToken").(string)
	if !valid || token == "" {
		return "", false
	}
	var user model.User
	err := h.db.Model(&model.User{}).Select("users.id").
		Joins("JOIN admin_sessions ON admin_sessions.user_id = users.id").
		Where("users.kakao_user_id = ? AND users.role = ? AND admin_sessions.token = ? AND admin_sessions.expires_at > ?", kakaoID, "operator", token, time.Now().UTC()).Scan(&user).Error
	if err != nil || user.ID == 0 {
		return "", false
	}
	return "operator:" + strconv.FormatUint(uint64(user.ID), 10), false
}

func (h *AdminHandler) denyAccess(c *gin.Context) {
	status := http.StatusUnauthorized
	if _, _, valid := getValidUser(sessions.Default(c)); valid {
		status = http.StatusForbidden
	}
	c.AbortWithStatusJSON(status, gin.H{"error": "이 기능을 사용할 권한이 없습니다."})
}

func (h *AdminHandler) requireOwner(c *gin.Context) {
	if !h.authenticated(c) {
		h.denyAccess(c)
		return
	}
	c.Next()
}

func (h *AdminHandler) Users(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	rows, total, err := h.svc.Users(c.Query("search"), page)
	if err != nil {
		managementError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": total, "page": page, "page_size": 50})
}

func (h *AdminHandler) SetUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		managementError(c, service.ErrManagementInput)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var input struct {
		Role string `json:"role"`
	}
	if c.ShouldBindJSON(&input) != nil {
		managementError(c, service.ErrManagementInput)
		return
	}
	if err := h.svc.SetUserRole(uint(id), input.Role, h.username()); err != nil {
		managementError(c, err)
		return
	}
	c.Status(204)
}

func (h *AdminHandler) UserRoleAudit(c *gin.Context) {
	rows, err := h.svc.UserRoleAudit()
	if err != nil {
		managementError(c, err)
		return
	}
	c.JSON(200, rows)
}
