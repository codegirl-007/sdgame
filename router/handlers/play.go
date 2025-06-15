package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"systemdesigngame/internals/auth"
	"systemdesigngame/internals/level"
)

type PlayHandler struct {
	Tmpl *template.Template
}

func (h *PlayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	levelName := strings.TrimPrefix(r.URL.Path, "/play/")
	levelName, err := url.PathUnescape(levelName)
	if err != nil {
		http.Error(w, "Invalid level name", http.StatusBadRequest)
		return
	}

	username := r.Context().Value(auth.UserLoginKey).(string)
	avatar := r.Context().Value(auth.UserAvatarKey).(string)

	lvl, err := level.GetLevel(strings.ToLower(levelName), level.DifficultyEasy)
	if err != nil {
		http.Error(w, "Level not found: "+err.Error(), http.StatusNotFound)
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

	h.Tmpl.ExecuteTemplate(w, "game.html", data)
}
