package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type ResultHandler struct {
	Tmpl *template.Template
}

type SuccessData struct {
	LevelName     string
	Score         int
	TargetRPS     int
	AchievedRPS   int
	TargetLatency int
	ActualLatency float64
	Availability  float64
	Feedback      []string
	LevelID       string
}

type FailureData struct {
	LevelName     string
	Reason        string
	TargetRPS     int
	AchievedRPS   int
	TargetLatency int
	ActualLatency float64
	TargetAvail   float64
	ActualAvail   float64
	FailedReqs    []string
	LevelID       string
}

func (r *ResultHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Default to success page for backward compatibility
	data := SuccessData{
		LevelName:     "Demo Level",
		Score:         85,
		TargetRPS:     10000,
		AchievedRPS:   10417,
		TargetLatency: 200,
		ActualLatency: 87,
		Availability:  99.9,
		Feedback:      []string{"All requirements met successfully!"},
		LevelID:       "demo",
	}

	if err := r.Tmpl.ExecuteTemplate(w, "success.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

type SuccessHandler struct {
	Tmpl *template.Template
}

func (h *SuccessHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := SuccessData{
		LevelName:     req.URL.Query().Get("level"),
		Score:         parseInt(req.URL.Query().Get("score"), 85),
		TargetRPS:     parseInt(req.URL.Query().Get("targetRPS"), 10000),
		AchievedRPS:   parseInt(req.URL.Query().Get("achievedRPS"), 10417),
		TargetLatency: parseInt(req.URL.Query().Get("targetLatency"), 200),
		ActualLatency: parseFloat(req.URL.Query().Get("actualLatency"), 87),
		Availability:  parseFloat(req.URL.Query().Get("availability"), 99.9),
		Feedback:      parseStringSlice(req.URL.Query().Get("feedback")),
		LevelID:       req.URL.Query().Get("levelId"),
	}

	if data.LevelName == "" {
		data.LevelName = "System Design Challenge"
	}
	if len(data.Feedback) == 0 {
		data.Feedback = []string{"All requirements met successfully!"}
	}

	if err := h.Tmpl.ExecuteTemplate(w, "success.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

type FailureHandler struct {
	Tmpl *template.Template
}

func (h *FailureHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := FailureData{
		LevelName:     req.URL.Query().Get("level"),
		Reason:        req.URL.Query().Get("reason"),
		TargetRPS:     parseInt(req.URL.Query().Get("targetRPS"), 10000),
		AchievedRPS:   parseInt(req.URL.Query().Get("achievedRPS"), 2847),
		TargetLatency: parseInt(req.URL.Query().Get("targetLatency"), 200),
		ActualLatency: parseFloat(req.URL.Query().Get("actualLatency"), 1247),
		TargetAvail:   parseFloat(req.URL.Query().Get("targetAvail"), 99.9),
		ActualAvail:   parseFloat(req.URL.Query().Get("actualAvail"), 87.3),
		FailedReqs:    parseStringSlice(req.URL.Query().Get("failedReqs")),
		LevelID:       req.URL.Query().Get("levelId"),
	}

	if data.LevelName == "" {
		data.LevelName = "System Design Challenge"
	}
	if data.Reason == "" {
		data.Reason = "performance"
	}
	if len(data.FailedReqs) == 0 {
		data.FailedReqs = []string{"Latency exceeded target", "Availability below requirement"}
	}

	if err := h.Tmpl.ExecuteTemplate(w, "failure.html", data); err != nil {
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

// Helper functions
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultValue
}

func parseFloat(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return defaultValue
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	// Split by pipe character for multiple values
	return strings.Split(s, "|")
}
