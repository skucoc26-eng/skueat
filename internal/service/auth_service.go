package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"restaurant-api/internal/config"
	"restaurant-api/internal/model"
)

// AuthService 카카오 소셜 인증 비즈니스 로직 인터페이스
type AuthService interface {
	GetAuthURL(state string) string
	GetToken(code string) (*model.KakaoTokenResponse, error)
	GetUserInfo(token string) (*model.KakaoUserResponse, error)
}

type authService struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewAuthService AuthService 구현체 생성
func NewAuthService(cfg *config.Config) AuthService {
	return &authService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *authService) GetAuthURL(state string) string {
	redirectURI := s.cfg.AppDomain + "/auth/kakao/callback"
	authURL := fmt.Sprintf(
		"https://kauth.kakao.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code",
		s.cfg.RestAPIKey,
		url.QueryEscape(redirectURI),
	)
	if state != "" {
		authURL += "&state=" + url.QueryEscape(state)
	}
	return authURL
}

func (s *authService) GetToken(code string) (*model.KakaoTokenResponse, error) {
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("client_id", s.cfg.RestAPIKey)
	params.Add("redirect_uri", s.cfg.AppDomain+"/auth/kakao/callback")
	params.Add("code", code)

	resp, err := s.httpClient.PostForm("https://kauth.kakao.com/oauth/token", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kakao token request returned status: %d", resp.StatusCode)
	}

	var tokenRes model.KakaoTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	return &tokenRes, nil
}

func (s *authService) GetUserInfo(token string) (*model.KakaoUserResponse, error) {
	req, err := http.NewRequest("GET", "https://kapi.kakao.com/v2/user/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kakao user info request returned status: %d", resp.StatusCode)
	}

	var userRes model.KakaoUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userRes); err != nil {
		return nil, err
	}
	return &userRes, nil
}
