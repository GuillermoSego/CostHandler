package main

// main es el punto de entrada de Go — como int main() en C.
// Siempre es package main y func main().

import (
	"log"
	"net/http"

	"github.com/GuillermoSego/costhandler/agent/agent"
	openaiClient "github.com/GuillermoSego/costhandler/agent/openai"
	"github.com/GuillermoSego/costhandler/mcp/config"
	"github.com/GuillermoSego/costhandler/mcp/database"
	"github.com/GuillermoSego/costhandler/mcp/handler"
	"github.com/GuillermoSego/costhandler/mcp/repository"
	"github.com/GuillermoSego/costhandler/mcp/service"

	"github.com/GuillermoSego/costhandler/bot/bot"
)

func main() {
	// ========== 1. CONFIGURACIÓN ==========
	// Carga las env vars (TELEGRAM_TOKEN, OPENAI_API_KEY, DB_PATH, SERVER_PORT)
	cfg := config.NewConfig()

	// Validamos que las env vars críticas existan
	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN no está configurado")
	}
	if cfg.OpenAIKey == "" {
		log.Fatal("OPENAI_API_KEY no está configurado")
	}

	// ========== 2. CAPA DE DATOS (MCP) ==========
	// Abrimos SQLite — si el archivo no existe, lo crea automáticamente
	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Error abriendo DB: %v", err)
	}

	// Creamos las tablas si no existen
	if err := database.CreateTables(db); err != nil {
		log.Fatalf("Error creando tablas: %v", err)
	}

	// Construimos la cadena: repository → service → handler
	// Cada pieza recibe la anterior — dependency injection manual
	repo := repository.NewSQLiteExpenseRepository(db)
	svc := service.NewExpenseService(repo)

	budgetRepo := repository.NewSQLiteBudgetRepository(db)
	budgetSvc := service.NewBudgetService(budgetRepo)

	expenseHandler := handler.NewExpenseHandler(svc, budgetSvc)

	// ========== 3. HTTP SERVER (API JSON + DASHBOARD) ==========
	mux := http.NewServeMux()
	expenseHandler.RegisterRoutes(mux)

	dashHandler, err := handler.NewDashboardHandler()
	if err != nil {
		log.Fatalf("Error creando dashboard handler: %v", err)
	}
	dashHandler.RegisterRoutes(mux)

	// ========== 4. AGENT (OpenAI) ==========
	// Cliente de OpenAI → Agent que clasifica mensajes
	classifier := openaiClient.NewClient(cfg.OpenAIKey)
	expenseAgent := agent.NewAgent(classifier)

	// ========== 5. BOT DE TELEGRAM ==========
	telegramBot, err := bot.NewBot(cfg.TelegramToken, expenseAgent, cfg.BaseURL, svc, budgetSvc)
	if err != nil {
		log.Fatalf("Error creando bot: %v", err)
	}

	// ========== 6. ARRANCAR TODO ==========
	// "go" lanza el bot en una goroutine (hilo paralelo).
	// Sin "go", el bot bloquearía aquí y el HTTP server nunca arrancaría.
	go telegramBot.Start()
	telegramBot.StartWeeklyScheduler()

	// El HTTP server corre en el hilo principal.
	// ListenAndServe también bloquea — se queda escuchando para siempre.
	log.Printf("HTTP server en http://localhost:%s", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, mux))
}
