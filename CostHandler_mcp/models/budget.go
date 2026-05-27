package models

type Budget struct {
	ID        int64   `json:"id"`
	User      string  `json:"user"`
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	UpdatedAt string  `json:"updated_at"`
}

type BudgetComparison struct {
	Category   string  `json:"category"`
	Budgeted   float64 `json:"budgeted"`
	Spent      float64 `json:"spent"`
	Remaining  float64 `json:"remaining"`
	Percentage float64 `json:"percentage"`
}

type UserChat struct {
	User   string `json:"user"`
	ChatID int64  `json:"chat_id"`
}
