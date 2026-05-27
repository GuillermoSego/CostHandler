package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// NewDB abre una conexión a SQLite y verifica que funcione.
func NewDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Ping verifica que la conexión realmente funciona
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// La query es una constante — no cambia nunca, así que la definimos fuera de la función.
const createExpensesTable = `
	CREATE TABLE IF NOT EXISTS expenses (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user        TEXT NOT NULL,
		amount      REAL NOT NULL,
		description TEXT NOT NULL,
		category    TEXT NOT NULL,
		raw_message TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)
`

const createBudgetsTable = `
	CREATE TABLE IF NOT EXISTS budgets (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		user       TEXT NOT NULL,
		category   TEXT NOT NULL,
		amount     REAL NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user, category)
	)
`

const createUserChatsTable = `
	CREATE TABLE IF NOT EXISTS user_chats (
		user    TEXT PRIMARY KEY,
		chat_id INTEGER NOT NULL
	)
`

func CreateTables(db *sql.DB) error {
	for _, query := range []string{createExpensesTable, createBudgetsTable, createUserChatsTable} {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
