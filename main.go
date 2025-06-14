package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"systemdesigngame/internals/level"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var jwtSecret []byte

var tmpl = template.Must(template.ParseGlob("static/*.html"))

type contextKey string

const (
	userIDKey     = contextKey("userID")
	userLoginKey  = contextKey("userLogin")
	userAvatarKey = contextKey("userAvatar")
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("failed to load .env")
	}

	jwtSecret = []byte(os.Getenv("JWT_SECRET"))

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", index)
	mux.HandleFunc("/play/", requireAuth(play))
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/callback", callbackHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	levels, err := level.LoadLevels("data/levels.json")
	if err != nil {
		panic("failed to load levels: " + err.Error())
	}
	level.InitRegistry(levels)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Server failed: " + err.Error())
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)
}

func index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title string
	}{
		Title: "Title",
	}

	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func play(w http.ResponseWriter, r *http.Request) {
	levelName := r.URL.Path[len("/play/"):]
	levelName, err := url.PathUnescape(levelName)
	avatar := r.Context().Value(userAvatarKey).(string)
	username := r.Context().Value(userLoginKey).(string)
	if err != nil {
		http.Error(w, "Invalid level name", http.StatusBadRequest)
		return
	}

	lvl, err := level.GetLevel(strings.ToLower(levelName), level.DifficultyEasy)
	if err != nil {
		http.Error(w, "Level not found"+err.Error(), http.StatusNotFound)
		return
	}
	allLevels := level.AllLevels()
	data := struct {
		Levels   []level.Level
		Level    *level.Level
		Avatar   string
		Username string
	}{
		Levels:   allLevels,
		Level:    lvl,
		Avatar:   avatar,
		Username: username,
	}

	tmpl.ExecuteTemplate(w, "game.html", data)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
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

func callbackHandler(w http.ResponseWriter, r *http.Request) {
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
	jwtToken, err := generateJWT(fmt.Sprintf("%d", userInfo.ID), userInfo.Login, userInfo.AvatarUrl)
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

func generateJWT(userID string, login string, avatarUrl string) (string, error) {
	claims := jwt.MapClaims{
		"sub":    userID,
		"login":  login,
		"avatar": avatarUrl,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
		"iat":    time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
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

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userLoginKey, login)
		ctx = context.WithValue(ctx, userAvatarKey, avatar)
		next(w, r.WithContext(ctx))
	}
}
