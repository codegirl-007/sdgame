package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"systemdesigngame/internal/auth"
	"systemdesigngame/internal/level"
)

type PlayHandler struct {
	Tmpl *template.Template
}

func (h *PlayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	levelId := r.PathValue("levelId")

	username := r.Context().Value(auth.UserLoginKey).(string)
	avatar := r.Context().Value(auth.UserAvatarKey).(string)
	lvl, err := level.GetLevelByID(levelId)
	if err != nil {
		http.Error(w, "Level not found: "+err.Error(), http.StatusNotFound)
		return
	}

	levelPayload, err := json.Marshal(lvl)

	if err != nil {
		fmt.Printf("error marshaling level: %v", err)
	}

	allLevels := level.AllLevels()
	data := struct {
		LevelPayload template.JS
		Levels       []level.Level
		Level        *level.Level
		Avatar       string
		Username     string
	}{
		LevelPayload: template.JS(levelPayload),
		Levels:       allLevels,
		Level:        lvl,
		Avatar:       avatar,
		Username:     username,
	}

	h.Tmpl.ExecuteTemplate(w, "game.html", data)
}
