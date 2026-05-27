package repository

import (
	"database/sql"

	"github.com/GuillermoSego/costhandler/mcp/models"
)

type BudgetRepository interface {
	Upsert(budget *models.Budget) error
	ListByUser(user string) ([]models.Budget, error)
	Delete(user, category string) error
	UpsertChatID(user string, chatID int64) error
	ListChatIDs() ([]models.UserChat, error)
}

type SQLiteBudgetRepository struct {
	db *sql.DB
}

func NewSQLiteBudgetRepository(db *sql.DB) *SQLiteBudgetRepository {
	return &SQLiteBudgetRepository{db: db}
}

func (r *SQLiteBudgetRepository) Upsert(budget *models.Budget) error {
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO budgets (user, category, amount) VALUES (?, ?, ?)",
		budget.User, budget.Category, budget.Amount,
	)
	return err
}

func (r *SQLiteBudgetRepository) ListByUser(user string) ([]models.Budget, error) {
	rows, err := r.db.Query(
		"SELECT id, user, category, amount, updated_at FROM budgets WHERE user = ? ORDER BY category",
		user,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []models.Budget
	for rows.Next() {
		var b models.Budget
		if err := rows.Scan(&b.ID, &b.User, &b.Category, &b.Amount, &b.UpdatedAt); err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, nil
}

func (r *SQLiteBudgetRepository) Delete(user, category string) error {
	_, err := r.db.Exec("DELETE FROM budgets WHERE user = ? AND category = ?", user, category)
	return err
}

func (r *SQLiteBudgetRepository) UpsertChatID(user string, chatID int64) error {
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO user_chats (user, chat_id) VALUES (?, ?)",
		user, chatID,
	)
	return err
}

func (r *SQLiteBudgetRepository) ListChatIDs() ([]models.UserChat, error) {
	rows, err := r.db.Query("SELECT user, chat_id FROM user_chats")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.UserChat
	for rows.Next() {
		var c models.UserChat
		if err := rows.Scan(&c.User, &c.ChatID); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, nil
}
