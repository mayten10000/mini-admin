package handlers

import (
	"database/sql"
	"net/http"

	"mini-admin/internal/ai"
	"mini-admin/internal/models"
	"mini-admin/internal/utils"
)

type AIHandler struct {
	DB       *sql.DB
	Analyzer *ai.Analyzer
	MaxUsers int
}

type aiAnalyzeResponse struct {
	Results []ai.AnalysisResult `json:"results"`
	Total   int                 `json:"total"`
	Model   string              `json:"model"`
}

func (h *AIHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if h.Analyzer == nil || !h.Analyzer.Configured() {
		utils.ErrorJSON(w, http.StatusServiceUnavailable, "AI is not configured: set OPENROUTER_API_KEY")
		return
	}

	limit := h.MaxUsers
	if limit <= 0 {
		limit = 100
	}
	users, err := models.ListUsersForAI(h.DB, limit)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to load users")
		return
	}

	results, err := h.Analyzer.Analyze(r.Context(), users)
	if err != nil {
		utils.ErrorJSON(w, http.StatusBadGateway, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, aiAnalyzeResponse{
		Results: results,
		Total:   len(results),
		Model:   h.Analyzer.Model,
	})
}
