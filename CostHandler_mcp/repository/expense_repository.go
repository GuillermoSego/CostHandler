package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/timeutil"
)

type ExpenseRepository interface {
	Create(expense *models.Expense) error
	CreateBatch(expenses []*models.Expense) error
	List(user string) ([]models.Expense, error)
	Update(expense *models.Expense) error
	Delete(id int64) error
	DeleteByGroup(groupID string) error
	ListFiltered(filter models.ExpenseFilter) ([]models.Expense, error)
	SumByCategory(user, from, to string) ([]models.CategorySummary, error)
	SumByDay(user, from, to string) ([]models.DailySummary, error)
	SumByMonth(user string, months int) ([]models.MonthlySummary, error)
	ListDistinctUsers() ([]string, error)
}

type SQLiteExpenseRepository struct {
	db *sql.DB
}

func NewSQLiteExpenseRepository(db *sql.DB) *SQLiteExpenseRepository {
	return &SQLiteExpenseRepository{db: db}
}

func (r *SQLiteExpenseRepository) Create(expense *models.Expense) error {
	if expense.CreatedAt != "" {
		_, err := r.db.Exec(
			`INSERT INTO expenses (user, amount, description, category, raw_message, created_at, installment_group, installment_number, total_installments)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			expense.User, expense.Amount, expense.Description, expense.Category.Name,
			expense.RawMessage, expense.CreatedAt,
			expense.InstallmentGroup, expense.InstallmentNumber, expense.TotalInstallments,
		)
		return err
	}
	_, err := r.db.Exec(
		`INSERT INTO expenses (user, amount, description, category, raw_message, installment_group, installment_number, total_installments)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		expense.User, expense.Amount, expense.Description, expense.Category.Name,
		expense.RawMessage,
		expense.InstallmentGroup, expense.InstallmentNumber, expense.TotalInstallments,
	)
	return err
}

func (r *SQLiteExpenseRepository) CreateBatch(expenses []*models.Expense) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("comenzando transacción: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO expenses (user, amount, description, category, raw_message, created_at, installment_group, installment_number, total_installments)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("preparando statement: %w", err)
	}
	defer stmt.Close()

	for _, e := range expenses {
		_, err := stmt.Exec(
			e.User, e.Amount, e.Description, e.Category.Name,
			e.RawMessage, e.CreatedAt,
			e.InstallmentGroup, e.InstallmentNumber, e.TotalInstallments,
		)
		if err != nil {
			return fmt.Errorf("insertando gasto: %w", err)
		}
	}

	return tx.Commit()
}

func (r *SQLiteExpenseRepository) DeleteByGroup(groupID string) error {
	_, err := r.db.Exec("DELETE FROM expenses WHERE installment_group = ?", groupID)
	return err
}

func (r *SQLiteExpenseRepository) List(user string) ([]models.Expense, error) {
	query := `SELECT id, user, amount, description, category, raw_message, created_at,
	                 installment_group, installment_number, total_installments
	          FROM expenses`
	var args []any
	if user != "" {
		query += " WHERE user = ?"
		args = append(args, user)
	}
	query += " ORDER BY created_at DESC"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense

	for rows.Next() {
		var expense models.Expense
		var categoryName string

		err := rows.Scan(
			&expense.ID,
			&expense.User,
			&expense.Amount,
			&expense.Description,
			&categoryName,
			&expense.RawMessage,
			&expense.CreatedAt,
			&expense.InstallmentGroup,
			&expense.InstallmentNumber,
			&expense.TotalInstallments,
		)
		if err != nil {
			return nil, err
		}

		expense.Category = models.Category{Name: categoryName}

		expenses = append(expenses, expense)
	}

	return expenses, nil
}

func (r *SQLiteExpenseRepository) Update(expense *models.Expense) error {
	_, err := r.db.Exec(
		`UPDATE expenses SET description = ?, category = ?, created_at = ? WHERE id = ?`,
		expense.Description, expense.Category.Name, expense.CreatedAt, expense.ID,
	)
	return err
}

// Delete elimina un gasto por su ID. Solo necesitas el ID, nada más.
func (r *SQLiteExpenseRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		return err
	}

	return nil
}

// dateRange convierte un período ("week", "month", "year") en fechas ISO.
func dateRange(period string) (string, string) {
	now := timeutil.Now()
	to := now.Format("2006-01-02")

	var from time.Time
	switch period {
	case "week":
		from = now.AddDate(0, 0, -7)
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	return from.Format("2006-01-02"), to
}

func (r *SQLiteExpenseRepository) ListFiltered(filter models.ExpenseFilter) ([]models.Expense, error) {
	from := filter.From
	to := filter.To
	if from == "" || to == "" {
		from, to = dateRange(filter.Period)
	}

	query := `SELECT id, user, amount, description, category, raw_message, created_at,
	                 installment_group, installment_number, total_installments
	          FROM expenses WHERE created_at >= ? AND created_at <= datetime(?, '+1 day')`
	args := []any{from, to}

	if filter.User != "" {
		query += " AND user = ?"
		args = append(args, filter.User)
	}

	if filter.Category != "" {
		query += " AND category = ?"
		args = append(args, filter.Category)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var expense models.Expense
		var categoryName string
		err := rows.Scan(
			&expense.ID,
			&expense.User,
			&expense.Amount,
			&expense.Description,
			&categoryName,
			&expense.RawMessage,
			&expense.CreatedAt,
			&expense.InstallmentGroup,
			&expense.InstallmentNumber,
			&expense.TotalInstallments,
		)
		if err != nil {
			return nil, err
		}
		expense.Category = models.Category{Name: categoryName}
		expenses = append(expenses, expense)
	}
	return expenses, nil
}

func (r *SQLiteExpenseRepository) SumByCategory(user, from, to string) ([]models.CategorySummary, error) {
	query := `SELECT category, SUM(amount) as total, COUNT(*) as count
	          FROM expenses
	          WHERE created_at >= ? AND created_at <= datetime(?, '+1 day')`
	args := []any{from, to}
	if user != "" {
		query += " AND user = ?"
		args = append(args, user)
	}
	query += " GROUP BY category ORDER BY total DESC"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.CategorySummary
	for rows.Next() {
		var s models.CategorySummary
		if err := rows.Scan(&s.Category, &s.Total, &s.Count); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *SQLiteExpenseRepository) SumByDay(user, from, to string) ([]models.DailySummary, error) {
	query := `SELECT DATE(created_at) as date, SUM(amount) as total
	          FROM expenses
	          WHERE created_at >= ? AND created_at <= datetime(?, '+1 day')`
	args := []any{from, to}
	if user != "" {
		query += " AND user = ?"
		args = append(args, user)
	}
	query += " GROUP BY DATE(created_at) ORDER BY date ASC"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.DailySummary
	for rows.Next() {
		var s models.DailySummary
		if err := rows.Scan(&s.Date, &s.Total); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *SQLiteExpenseRepository) SumByMonth(user string, months int) ([]models.MonthlySummary, error) {
	cutoff := timeutil.Now().AddDate(0, -months, 0).Format("2006-01-02")
	query := `SELECT STRFTIME('%Y-%m', created_at) as month, SUM(amount) as total
	          FROM expenses
	          WHERE created_at >= ?`
	args := []any{cutoff}
	if user != "" {
		query += " AND user = ?"
		args = append(args, user)
	}
	query += " GROUP BY STRFTIME('%Y-%m', created_at) ORDER BY month ASC"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.MonthlySummary
	for rows.Next() {
		var s models.MonthlySummary
		if err := rows.Scan(&s.Month, &s.Total); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *SQLiteExpenseRepository) ListDistinctUsers() ([]string, error) {
	rows, err := r.db.Query("SELECT DISTINCT user FROM expenses ORDER BY user")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
