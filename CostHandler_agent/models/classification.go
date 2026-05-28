package models

type ClassificationResult struct {
	Amount       float64 `json:"amount"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	Confidence   float64 `json:"confidence"`
	Installments int     `json:"installments"`
	Date         string  `json:"date"`
}
