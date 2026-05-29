package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	ErrCMDBAuthDisabled       = errors.New("cmdb auth disabled")
	ErrCMDBAuthConfig         = errors.New("cmdb auth config invalid")
	ErrCMDBAuthTokenInvalid   = errors.New("cmdb auth token invalid")
	ErrCMDBAuthUserNotMapped  = errors.New("cmdb auth user not mapped")
	ErrCMDBAuthUserInfoFailed = errors.New("cmdb auth userinfo failed")
)

type cmdbJWTClaims struct {
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	UID      any    `json:"uid,omitempty"`
	jwt.RegisteredClaims
}

type cmdbUserInfoResponse struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Name     string `json:"name"`
	UID      any    `json:"uid"`
}

func CMDBAuthEnabled() bool {
	return common.GetEnvOrDefaultBool("CMDB_AUTH_ENABLED", false)
}

func getCMDBJWTSecret() string {
	if secret := strings.TrimSpace(os.Getenv("CMDB_JWT_SECRET")); secret != "" {
		return secret
	}
	return strings.TrimSpace(os.Getenv("CMDB_SECRET_KEY"))
}

func parseBearerToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func ExtractCMDBAccessToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	headerName := strings.TrimSpace(common.GetEnvOrDefaultString("CMDB_AUTH_HEADER", "Access-Token"))
	if headerName != "" {
		if token := parseBearerToken(r.Header.Get(headerName)); token != "" {
			return token
		}
	}
	if common.GetEnvOrDefaultBool("CMDB_AUTH_ALLOW_AUTHORIZATION", true) {
		token := parseBearerToken(r.Header.Get("Authorization"))
		if strings.Count(token, ".") == 2 {
			return token
		}
	}
	return ""
}

func AuthenticateCMDBAccessToken(ctx context.Context, token string) (*model.User, error) {
	if !CMDBAuthEnabled() {
		return nil, ErrCMDBAuthDisabled
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrCMDBAuthTokenInvalid
	}
	secret := getCMDBJWTSecret()
	if secret == "" {
		return nil, fmt.Errorf("%w: CMDB_JWT_SECRET is empty", ErrCMDBAuthConfig)
	}

	claims := &cmdbJWTClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithLeeway(30*time.Second))
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: %v", ErrCMDBAuthTokenInvalid, err)
	}

	identity := cmdbUserInfoResponse{
		Email:    strings.TrimSpace(claims.Email),
		Username: strings.TrimSpace(claims.Username),
		UID:      claims.UID,
	}
	if claims.Subject != "" && identity.Email == "" {
		identity.Email = strings.TrimSpace(claims.Subject)
	}

	if userInfoURL := strings.TrimSpace(os.Getenv("CMDB_AUTH_USERINFO_URL")); userInfoURL != "" {
		userInfo, err := fetchCMDBUserInfo(ctx, userInfoURL, token)
		if err != nil {
			return nil, err
		}
		if userInfo.Email != "" {
			identity.Email = strings.TrimSpace(userInfo.Email)
		}
		if userInfo.Username != "" {
			identity.Username = strings.TrimSpace(userInfo.Username)
		}
		if userInfo.UID != nil {
			identity.UID = userInfo.UID
		}
	}

	return findMappedNewAPIUser(identity)
}

func fetchCMDBUserInfo(ctx context.Context, userInfoURL string, token string) (*cmdbUserInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCMDBAuthUserInfoFailed, err)
	}
	req.Header.Set("Access-Token", token)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCMDBAuthUserInfoFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrCMDBAuthUserInfoFailed, resp.StatusCode)
	}

	var userInfo cmdbUserInfoResponse
	if err := common.DecodeJson(resp.Body, &userInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCMDBAuthUserInfoFailed, err)
	}
	return &userInfo, nil
}

func findMappedNewAPIUser(identity cmdbUserInfoResponse) (*model.User, error) {
	var user model.User
	email := strings.TrimSpace(identity.Email)
	username := strings.TrimSpace(identity.Username)

	if email != "" {
		err := model.DB.Omit("password").Where("email = ?", email).First(&user).Error
		if err == nil {
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", model.ErrDatabase, err)
		}
	}

	if common.GetEnvOrDefaultBool("CMDB_AUTH_MATCH_USERNAME", false) && username != "" {
		err := model.DB.Omit("password").Where("username = ?", username).First(&user).Error
		if err == nil {
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %v", model.ErrDatabase, err)
		}
	}

	return nil, ErrCMDBAuthUserNotMapped
}
