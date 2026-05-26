package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/GuillermoSego/costhandler/agent/agent"
	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot tiene dos dependencias:
// - api: la conexión con Telegram (para recibir y enviar mensajes)
// - agent: el cerebro que clasifica gastos (Fase 2)
type Bot struct {
	api          *tgbotapi.BotAPI
	agent        *agent.Agent
	dashboardURL string
	svc          *service.ExpenseService
}

// NewBot crea la conexión con Telegram usando el token de BotFather.
// Si el token es inválido, Telegram rechaza y devuelve error.
func NewBot(token string, agent *agent.Agent, dashboardURL string, svc *service.ExpenseService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("connecting to telegram: %w", err)
	}

	log.Printf("Bot conectado como: @%s", api.Self.UserName)

	return &Bot{
		api:          api,
		agent:        agent,
		dashboardURL: dashboardURL,
		svc:          svc,
	}, nil
}

// Start inicia el loop principal del bot.
// Long polling = le preguntamos a Telegram "¿hay mensajes nuevos?" cada pocos segundos.
// Este método BLOQUEA — se queda corriendo hasta que lo detengas (Ctrl+C).
func (b *Bot) Start() {
	// Offset 0 = desde el último mensaje no procesado
	// Timeout 60 = espera hasta 60 segundos antes de responder vacío
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// GetUpdatesChan devuelve un CHANNEL — un tubo por donde llegan los mensajes.
	// Es como un stream/observable en JS o un generator en Python.
	updates := b.api.GetUpdatesChan(updateConfig)

	log.Println("Bot escuchando mensajes...")

	// for range sobre un channel = "procesa cada mensaje que llegue, para siempre"
	// Esto es el event loop del bot.
	for update := range updates {
		// Solo nos interesan mensajes de texto (ignoramos stickers, fotos, etc.)
		if update.Message == nil {
			continue
		}

		b.handleUpdate(update.Message)
	}
}

// handleUpdate decide qué hacer con cada mensaje.
// Si empieza con "/" es un comando, si no es un gasto.
func (b *Bot) handleUpdate(message *tgbotapi.Message) {
	user := message.From.UserName
	if user == "" {
		user = message.From.FirstName
	}
	log.Printf("Mensaje recibido de @%s: %s", user, message.Text)

	if message.IsCommand() {
		b.handleCommand(message)
	} else {
		b.handleExpense(message)
	}
}

// handleCommand procesa los comandos del bot (/start, /ayuda, etc.)
func (b *Bot) handleCommand(message *tgbotapi.Message) {
	var response string

	// message.Command() devuelve el comando SIN el "/" — "start", "ayuda", etc.
	user := message.From.UserName
	if user == "" {
		user = message.From.FirstName
	}
	log.Printf("Comando /%s de @%s", message.Command(), user)

	switch message.Command() {
	case "start":
		response = "¡Hola! Soy CostHandler, tu asistente de gastos.\n\n" +
			"Envíame un mensaje con tu gasto y yo lo clasifico.\n" +
			"Ejemplo: \"150 tacos al pastor\"\n\n" +
			"Comandos:\n" +
			"/resumen — resumen del mes\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/dashboard — ver dashboard de gastos\n" +
			"/ayuda — ver esta lista"
	case "ayuda":
		response = "Comandos disponibles:\n" +
			"/start — bienvenida\n" +
			"/resumen — resumen del mes\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/dashboard — ver dashboard de gastos\n" +
			"/ayuda — ver esta lista"
	case "resumen":
		b.handleResumen(message)
		return
	case "ultimos":
		b.handleUltimos(message)
		return
	case "dashboard":
		url := b.dashboardURL + "/dashboard"
		if strings.HasPrefix(b.dashboardURL, "https://") {
			b.sendMessageWithButton(message.Chat.ID, "Abre el dashboard para ver tus gastos.", "Abrir Dashboard", url)
		} else {
			b.sendMessage(message.Chat.ID, "Abre el dashboard para ver tus gastos:\n"+url)
		}
		return
	default:
		response = "Comando no reconocido. Usa /ayuda para ver los disponibles."
	}

	b.sendMessage(message.Chat.ID, response)
}

// handleExpense procesa mensajes libres — los manda al agent para clasificar.
func (b *Bot) handleExpense(message *tgbotapi.Message) {
	// Sacamos el username de Telegram como identificador de usuario
	user := message.From.UserName
	if user == "" {
		user = message.From.FirstName // Fallback si no tiene username
	}

	result, err := b.agent.ProcessMessage(context.Background(), user, message.Text)
	if err != nil {
		log.Printf("Error clasificando gasto de @%s: %v", user, err)
		b.sendMessage(message.Chat.ID, "No pude clasificar ese gasto: "+err.Error())
		return
	}
	log.Printf("Gasto clasificado: $%.2f %s (%s) — confianza: %.0f%%", result.Amount, result.Description, result.Category, result.Confidence*100)

	expense := &models.Expense{
		User:        user,
		Amount:      result.Amount,
		Description: result.Description,
		Category:    models.Category{Name: result.Category},
		RawMessage:  message.Text,
	}
	if err := b.svc.Create(expense); err != nil {
		log.Printf("Error guardando gasto de @%s: %v", user, err)
		b.sendMessage(message.Chat.ID, "Gasto clasificado pero no se pudo guardar: "+err.Error())
		return
	}
	log.Printf("Gasto guardado en DB para @%s", user)

	var sb strings.Builder
	sb.WriteString("Gasto registrado:\n\n")
	sb.WriteString(fmt.Sprintf("💰 Monto: $%.2f\n", result.Amount))
	sb.WriteString(fmt.Sprintf("📁 Categoría: %s\n", result.Category))
	sb.WriteString(fmt.Sprintf("📝 Descripción: %s\n", result.Description))
	sb.WriteString(fmt.Sprintf("🎯 Confianza: %.0f%%\n", result.Confidence*100))

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) handleResumen(message *tgbotapi.Message) {
	data, err := b.svc.GetDashboardData("month", "")
	if err != nil {
		log.Printf("Error obteniendo resumen: %v", err)
		b.sendMessage(message.Chat.ID, "Error obteniendo resumen: "+err.Error())
		return
	}

	if data.ExpenseCount == 0 {
		b.sendMessage(message.Chat.ID, "No hay gastos registrados este mes.")
		return
	}
	log.Printf("Resumen generado: %d gastos, $%.2f total", data.ExpenseCount, data.TotalAmount)

	var sb strings.Builder
	sb.WriteString("Resumen del mes:\n\n")
	sb.WriteString(fmt.Sprintf("💰 Total: $%.2f\n", data.TotalAmount))
	sb.WriteString(fmt.Sprintf("📊 Gastos: %d\n", data.ExpenseCount))
	sb.WriteString(fmt.Sprintf("📅 Promedio diario: $%.2f\n", data.DailyAverage))
	if data.TopCategory != "" {
		sb.WriteString(fmt.Sprintf("🏆 Categoría top: %s ($%.2f)\n", data.TopCategory, data.TopCategoryAmt))
	}

	if len(data.ByCategory) > 0 {
		sb.WriteString("\nPor categoría:\n")
		for _, c := range data.ByCategory {
			sb.WriteString(fmt.Sprintf("  %s: $%.2f (%d)\n", c.Category, c.Total, c.Count))
		}
	}

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) handleUltimos(message *tgbotapi.Message) {
	filter := models.ExpenseFilter{Period: "month"}
	expenses, err := b.svc.ListFiltered(filter)
	if err != nil {
		log.Printf("Error obteniendo últimos gastos: %v", err)
		b.sendMessage(message.Chat.ID, "Error obteniendo gastos: "+err.Error())
		return
	}
	log.Printf("Últimos gastos: %d resultados", len(expenses))

	if len(expenses) == 0 {
		b.sendMessage(message.Chat.ID, "No hay gastos registrados este mes.")
		return
	}

	limit := 5
	if len(expenses) < limit {
		limit = len(expenses)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Últimos %d gastos:\n\n", limit))
	for i, e := range expenses[:limit] {
		date := e.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}
		sb.WriteString(fmt.Sprintf("%d. $%.2f — %s (%s) — %s\n", i+1, e.Amount, e.Description, e.Category.Name, date))
	}

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error enviando mensaje: %v", err)
	}
}

func (b *Bot) sendMessageWithButton(chatID int64, text, buttonLabel, url string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(buttonLabel, url),
		),
	)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error enviando mensaje: %v", err)
	}
}
