package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/GuillermoSego/costhandler/mcp/models"
)

type ExpenseRepository interface {
	Create(expense *models.Expense) error
	CreateBatch(expenses []*models.Expense) error
	List() ([]models.Expense, error)
	Delete(id int64) error
	DeleteByGroup(groupID string) error
	ListFiltered(filter models.ExpenseFilter) ([]models.Expense, error)
	SumByCategory(from, to string) ([]models.CategorySummary, error)
	SumByDay(from, to string) ([]models.DailySummary, error)
	SumByMonth(months int) ([]models.MonthlySummary, error)
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

// List devuelve todos los gastos.
func (r *SQLiteExpenseRepository) List() ([]models.Expense, error) {
	// Columnas explícitas — si usas SELECT * y mañana agregas una columna, Scan se rompe.
	rows, err := r.db.Query(
		`SELECT id, user, amount, description, category, raw_message, created_at,
		        installment_group, installment_number, total_installments
		 FROM expenses ORDER BY created_at DESC`,
	)
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
	now := time.Now()
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

func (r *SQLiteExpenseRepository) SumByCategory(from, to string) ([]models.CategorySummary, error) {
	rows, err := r.db.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count
		 FROM expenses
		 WHERE created_at >= ? AND created_at <= datetime(?, '+1 day')
		 GROUP BY category
		 ORDER BY total DESC`,
		from, to,
	)
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

func (r *SQLiteExpenseRepository) SumByDay(from, to string) ([]models.DailySummary, error) {
	rows, err := r.db.Query(
		`SELECT DATE(created_at) as date, SUM(amount) as total
		 FROM expenses
		 WHERE created_at >= ? AND created_at <= datetime(?, '+1 day')
		 GROUP BY DATE(created_at)
		 ORDER BY date ASC`,
		from, to,
	)
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

func (r *SQLiteExpenseRepository) SumByMonth(months int) ([]models.MonthlySummary, error) {
	rows, err := r.db.Query(
		`SELECT STRFTIME('%Y-%m', created_at) as month, SUM(amount) as total
		 FROM expenses
		 WHERE created_at >= DATE('now', ?)
		 GROUP BY STRFTIME('%Y-%m', created_at)
		 ORDER BY month ASC`,
		fmt.Sprintf("-%d months", months),
	)
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
