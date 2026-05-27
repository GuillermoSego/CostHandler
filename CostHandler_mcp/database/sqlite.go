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
		id                  INTEGER PRIMARY KEY AUTOINCREMENT,
		user                TEXT NOT NULL,
		amount              REAL NOT NULL,
		description         TEXT NOT NULL,
		category            TEXT NOT NULL,
		raw_message         TEXT,
		created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
		installment_group   TEXT DEFAULT '',
		installment_number  INTEGER DEFAULT 0,
		total_installments  INTEGER DEFAULT 0
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

func RunMigrations(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(expenses)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasInstallmentGroup := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "installment_group" {
			hasInstallmentGroup = true
			break
		}
	}

	if !hasInstallmentGroup {
		migrations := []string{
			"ALTER TABLE expenses ADD COLUMN installment_group TEXT DEFAULT ''",
			"ALTER TABLE expenses ADD COLUMN installment_number INTEGER DEFAULT 0",
			"ALTER TABLE expenses ADD COLUMN total_installments INTEGER DEFAULT 0",
		}
		for _, m := range migrations {
			if _, err := db.Exec(m); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateTables(db *sql.DB) error {
	for _, query := range []string{createExpensesTable, createBudgetsTable, createUserChatsTable} {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
