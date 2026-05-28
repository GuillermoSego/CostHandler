package models

type ExpenseFilter struct {
	User     string `json:"user"`
	Period   string `json:"period"`
	Category string `json:"category"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type CategorySummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

type DailySummary struct {
	Date  string  `json:"date"`
	Total float64 `json:"total"`
}

type MonthlySummary struct {
	Month string  `json:"month"`
	Total float64 `json:"total"`
}

type DashboardData struct {
	TotalAmount      float64            `json:"total_amount"`
	DailyAverage     float64            `json:"daily_average"`
	TopCategory      string             `json:"top_category"`
	TopCategoryAmt   float64            `json:"top_category_amount"`
	PrevTotal        float64            `json:"prev_total"`
	ByCategory       []CategorySummary  `json:"by_category"`
	ByDay            []DailySummary     `json:"by_day"`
	ByMonth          []MonthlySummary   `json:"by_month"`
	ExpenseCount     int                `json:"expense_count"`
	BudgetComparison []BudgetComparison `json:"budget_comparison"`
	TotalBudgeted    float64            `json:"total_budgeted"`
}

type InstallmentGroupSummary struct {
	GroupID         string  `json:"group_id"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	TotalAmount     float64 `json:"total_amount"`
	PaidAmount      float64 `json:"paid_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
	PaidCount       int     `json:"paid_count"`
	TotalCount      int     `json:"total_count"`
	PerInstallment  float64 `json:"per_installment"`
	LastPaymentDate string  `json:"last_payment_date"`
	User            string  `json:"user"`
}

type InstallmentSummary struct {
	Groups           []InstallmentGroupSummary `json:"groups"`
	TotalRemaining   float64                   `json:"total_remaining"`
	DebtFreeDate     string                    `json:"debt_free_date"`
	ActiveGroupCount int                       `json:"active_group_count"`
}
