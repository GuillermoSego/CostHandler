package repository

import (
	"database/sql"

	"github.com/GuillermoSego/costhandler/mcp/models"
)

type ExpenseRepository interface {
	Create(expense *models.Expense) error
	List() ([]models.Expense, error)
	Delete(id int64) error
}

type SQLiteExpenseRepository struct {
	db *sql.DB
}

func NewSQLiteExpenseRepository(db *sql.DB) *SQLiteExpenseRepository {
	return &SQLiteExpenseRepository{db: db}
}

func (r *SQLiteExpenseRepository) Create(expense *models.Expense) error {
	_, err := r.db.Exec(
		"INSERT INTO expenses (user, amount, description, category, raw_message) VALUES (?, ?, ?, ?, ?)",
		expense.User,
		expense.Amount,
		expense.Description,
		expense.Category.Name,
		expense.RawMessage,
	)
	if err != nil {
		return err
	}

	return nil
}

// List devuelve todos los gastos.
func (r *SQLiteExpenseRepository) List() ([]models.Expense, error) {
	// Columnas explícitas — si usas SELECT * y mañana agregas una columna, Scan se rompe.
	rows, err := r.db.Query(
		"SELECT id, user, amount, description, category, raw_message, created_at FROM expenses ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	// defer = "ejecuta esto cuando la función termine". Cierra las filas para liberar recursos.
	defer rows.Close()

	// Creamos un slice vacío donde iremos agregando cada gasto.
	var expenses []models.Expense

	// rows.Next() mueve al siguiente registro. Devuelve false cuando no hay más.
	for rows.Next() {
		var expense models.Expense
		var categoryName string

		// Scan copia los valores de la fila actual a nuestras variables.
		// El orden DEBE coincidir con las columnas del SELECT.
		err := rows.Scan(
			&expense.ID,
			&expense.User,
			&expense.Amount,
			&expense.Description,
			&categoryName,
			&expense.RawMessage,
			&expense.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convertimos el string de la DB al struct Category de Go.
		expense.Category = models.Category{Name: categoryName}

		// append agrega un elemento al slice (como push en JS o append en Python).
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
