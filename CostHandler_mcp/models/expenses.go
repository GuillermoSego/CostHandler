package models

type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type Expense struct {
	ID          int64    `json:"id"`
	User        string   `json:"user"`
	Amount      float64  `json:"amount"`
	Description string   `json:"description"`
	Category    Category `json:"category"`
	RawMessage  string   `json:"raw_message"`
	CreatedAt   string   `json:"created_at"`
}
