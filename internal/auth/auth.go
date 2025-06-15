package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JwtSecret []byte

const (
	UserIDKey     = contextKey("userID")
	UserLoginKey  = contextKey("userLogin")
	UserAvatarKey = contextKey("userAvatar")
)

type contextKey string

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil && token.Valid {
			http.Redirect(w, r, "/play", http.StatusFound)
			return
		}
	}

	clientId := os.Getenv("GITHUB_CLIENT_ID")
	redirectURI := os.Getenv("GITHUB_CALLBACK")
	scope := "read:user user:email"
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s",
		clientId, url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
	)
	http.Redirect(w, r, url, http.StatusFound)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code in query", http.StatusBadRequest)
		return
	}

	data := url.Values{}
	data.Set("client_id", os.Getenv("GITHUB_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("GITHUB_CLIENT_SECRET"))
	data.Set("code", code)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		http.Error(w, "Failed to create token request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	json.NewDecoder(resp.Body).Decode(&tokenResp)

	// Use token to get user info
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}

	defer userResp.Body.Close()

	var userInfo struct {
		Login     string `json:"login"`
		ID        int    `json:"id"`
		AvatarUrl string `json:"avatar_url"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to parse user info", http.StatusInternalServerError)
		return
	}

	// Generate JWT
	fmt.Printf("generating jwt: %s, %s, %s\n", fmt.Sprintf("%d", userInfo.ID), userInfo.Login, userInfo.AvatarUrl)
	jwtToken, err := GenerateJWT(fmt.Sprintf("%d", userInfo.ID), userInfo.Login, userInfo.AvatarUrl)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set JWT in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set to true in production (HTTPS)
	})

	http.Redirect(w, r, "/play", http.StatusFound)
}

func GenerateJWT(userID string, login string, avatarUrl string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"login":  login,
		"avatar": avatarUrl,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		userID, ok1 := claims["sub"].(string)
		login, ok2 := claims["login"].(string)
		avatar, _ := claims["avatar"].(string)
		if !ok1 || !ok2 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserLoginKey, login)
		ctx = context.WithValue(ctx, UserAvatarKey, avatar)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
