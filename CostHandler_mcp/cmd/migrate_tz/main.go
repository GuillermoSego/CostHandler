package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./expenses.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("abriendo DB: %v", err)
	}
	defer db.Close()

	res, err := db.Exec("UPDATE expenses SET created_at = datetime(created_at, '-6 hours')")
	if err != nil {
		log.Fatalf("migrando expenses: %v", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("expenses actualizados: %d\n", n)

	res, err = db.Exec("UPDATE budgets SET updated_at = datetime(updated_at, '-6 hours')")
	if err != nil {
		log.Fatalf("migrando budgets: %v", err)
	}
	n, _ = res.RowsAffected()
	fmt.Printf("budgets actualizados: %d\n", n)

	fmt.Println("migración completada")
}
