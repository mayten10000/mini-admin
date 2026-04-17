package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mini-admin/internal/models"
)

type Analyzer struct {
	APIKey   string
	BaseURL  string
	Model    string
	Timeout  time.Duration
	MaxUsers int
}

type AnalysisResult struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	RiskLevel          string    `json:"risk_level"`
	Comment            string    `json:"comment"`
	RecommendedAction  string    `json:"recommended_action"`
}

type aiUserItem struct {
	ID                int64  `json:"id"`
	RiskLevel         string `json:"risk_level"`
	Comment           string `json:"comment"`
	RecommendedAction string `json:"recommended_action"`
}

type aiResponse struct {
	Users []aiUserItem `json:"users"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `Ты помощник администратора, делающий первичный риск-обзор пользователей админ-панели. ` +
	`По каждому пользователю ты должен оценить, выглядит ли он подозрительным/тестовым/некачественным, ` +
	`основываясь ТОЛЬКО на полях id, name, email, status, created_at, updated_at. ` +
	`Это не настоящий fraud detection, а быстрая эвристическая разметка для оператора.

Эвристики:
- email на одноразовых доменах (mailinator, tempmail, guerrillamail, 10minutemail, sharklasers, yopmail и т.п.) → high
- имя из 1 символа, цифр, "test", "asdf", "qwerty", повторяющихся букв → medium/high
- статус disabled → medium (либо подтвердить блокировку)
- email явно тестовый (test@, qa@, demo@) при подозрительном имени → high
- пользователь, у которого created_at == updated_at и обычные данные → low
- всё нормально → low

Верни СТРОГО JSON без markdown:
{"users":[{"id":<int>,"risk_level":"low|medium|high","comment":"<≤120 симв.>","recommended_action":"<≤80 симв.>"}, ...]}

Включи в ответ КАЖДОГО пользователя из входа, не пропуская. Поля комментария и действия — по-русски, кратко.`

func New(apiKey, baseURL, model string, timeout time.Duration, maxUsers int) *Analyzer {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if maxUsers <= 0 {
		maxUsers = 100
	}
	return &Analyzer{
		APIKey:   apiKey,
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Model:    model,
		Timeout:  timeout,
		MaxUsers: maxUsers,
	}
}

func (a *Analyzer) Configured() bool {
	return a != nil && a.APIKey != ""
}

func (a *Analyzer) Analyze(ctx context.Context, users []models.User) ([]AnalysisResult, error) {
	if !a.Configured() {
		return nil, errors.New("AI is not configured: set OPENROUTER_API_KEY")
	}
	if len(users) == 0 {
		return []AnalysisResult{}, nil
	}

	if len(users) > a.MaxUsers {
		users = users[:a.MaxUsers]
	}

	userPayload := make([]map[string]any, len(users))
	for i, u := range users {
		userPayload[i] = map[string]any{
			"id":         u.ID,
			"name":       u.Name,
			"email":      u.Email,
			"status":     u.Status,
			"created_at": u.CreatedAt.Format(time.RFC3339),
			"updated_at": u.UpdatedAt.Format(time.RFC3339),
		}
	}
	usersJSON, err := json.Marshal(userPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal users: %w", err)
	}

	userMsg := "Список пользователей (JSON):\n" + string(usersJSON) +
		"\n\nВерни JSON по схеме, описанной в инструкции."

	reqBody := chatRequest{
		Model: a.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		Temperature: 0.1,
		MaxTokens:   4096,
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{Type: "json_object"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(ctx, a.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost,
		a.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.APIKey)
	httpReq.Header.Set("X-Title", "mini-admin")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call AI: %w", err)
	}
	defer resp.Body.Close()

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode AI response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg := "AI request failed"
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return nil, fmt.Errorf("%s (status %d)", msg, resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("AI returned empty response")
	}

	content := parsed.Choices[0].Message.Content
	jsonStr := extractJSONObject(content)
	if jsonStr == "" {
		return nil, errors.New("AI did not return JSON")
	}

	var ai aiResponse
	if err := json.Unmarshal([]byte(jsonStr), &ai); err != nil {
		return nil, fmt.Errorf("parse AI JSON: %w", err)
	}

	byID := make(map[int64]aiUserItem, len(ai.Users))
	for _, item := range ai.Users {
		byID[item.ID] = item
	}

	out := make([]AnalysisResult, 0, len(users))
	for _, u := range users {
		item, ok := byID[u.ID]
		risk := normalizeRisk(item.RiskLevel)
		comment := strings.TrimSpace(item.Comment)
		action := strings.TrimSpace(item.RecommendedAction)
		if !ok {
			risk = "low"
			comment = "AI не вернул оценку для этого пользователя"
			action = "Проверить вручную"
		}
		out = append(out, AnalysisResult{
			ID:                u.ID,
			Name:              u.Name,
			Email:             u.Email,
			Status:            u.Status,
			CreatedAt:         u.CreatedAt,
			UpdatedAt:         u.UpdatedAt,
			RiskLevel:         risk,
			Comment:           comment,
			RecommendedAction: action,
		})
	}
	return out, nil
}

func normalizeRisk(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "low"
	}
}

func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
