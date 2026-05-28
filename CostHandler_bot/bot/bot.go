package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GuillermoSego/costhandler/agent/agent"
	agentModels "github.com/GuillermoSego/costhandler/agent/models"
	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/service"
	"github.com/GuillermoSego/costhandler/mcp/timeutil"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	agent         *agent.Agent
	dashboardURL  string
	svc           *service.ExpenseService
	budgetSvc     *service.BudgetService
	allowedUsers  map[string]bool
	pendingBudget map[int64]string
	mu            sync.Mutex
}

func NewBot(token string, agent *agent.Agent, dashboardURL string, svc *service.ExpenseService, budgetSvc *service.BudgetService, allowedUsers []string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("connecting to telegram: %w", err)
	}

	log.Printf("Bot conectado como: @%s", api.Self.UserName)

	allowed := make(map[string]bool, len(allowedUsers))
	for _, u := range allowedUsers {
		allowed[u] = true
	}

	return &Bot{
		api:           api,
		agent:         agent,
		dashboardURL:  dashboardURL,
		svc:           svc,
		budgetSvc:     budgetSvc,
		allowedUsers:  allowed,
		pendingBudget: make(map[int64]string),
	}, nil
}

func (b *Bot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	log.Println("Bot escuchando mensajes...")

	for update := range updates {
		if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		b.saveChatID(update.Message)
		b.handleUpdate(update.Message)
	}
}

func (b *Bot) StartWeeklyScheduler() {
	go func() {
		for {
			next := nextWeekday(time.Monday, 9, 0)
			log.Printf("Próximo resumen semanal: %s", next.Format("2006-01-02 15:04"))
			time.Sleep(time.Until(next))
			b.sendWeeklySummaries()
		}
	}()
}

func (b *Bot) saveChatID(message *tgbotapi.Message) {
	user := getUser(message)
	if err := b.budgetSvc.UpsertChatID(user, message.Chat.ID); err != nil {
		log.Printf("Error guardando chat_id de @%s: %v", user, err)
	}
}

func (b *Bot) isAllowedUser(username string) bool {
	if len(b.allowedUsers) == 0 {
		return true
	}
	return b.allowedUsers[username]
}

func (b *Bot) handleUpdate(message *tgbotapi.Message) {
	user := getUser(message)
	log.Printf("Mensaje recibido de @%s: %s", user, message.Text)

	if !b.isAllowedUser(user) {
		log.Printf("Usuario no autorizado: @%s", user)
		b.sendMessage(message.Chat.ID, "No estás autorizado para usar este bot.")
		return
	}

	b.mu.Lock()
	category, pending := b.pendingBudget[message.Chat.ID]
	b.mu.Unlock()

	if pending {
		b.handleBudgetAmount(message, category)
		return
	}

	if message.IsCommand() {
		b.handleCommand(message)
	} else {
		b.handleExpense(message)
	}
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	user := getUser(message)
	log.Printf("Comando /%s de @%s", message.Command(), user)

	var response string

	switch message.Command() {
	case "start":
		response = "¡Hola! Soy CostHandler, tu asistente de gastos.\n\n" +
			"Envíame un mensaje con tu gasto y yo lo clasifico.\n" +
			"Ejemplo: \"150 tacos al pastor\"\n\n" +
			"Comandos:\n" +
			"/resumen — resumen del mes con insights\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/presupuesto — ver/editar presupuesto mensual\n" +
			"/dashboard — ver dashboard de gastos\n" +
			"/ayuda — ver esta lista"
	case "ayuda":
		response = "Comandos disponibles:\n" +
			"/start — bienvenida\n" +
			"/resumen — resumen del mes con insights\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/presupuesto — ver/editar presupuesto mensual\n" +
			"/dashboard — ver dashboard de gastos\n" +
			"/ayuda — ver esta lista"
	case "resumen":
		b.handleResumen(message)
		return
	case "ultimos":
		b.handleUltimos(message)
		return
	case "presupuesto":
		b.handlePresupuesto(message)
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

func (b *Bot) handleExpense(message *tgbotapi.Message) {
	user := getUser(message)

	result, err := b.agent.ProcessMessage(context.Background(), user, message.Text)
	if err != nil {
		log.Printf("Error clasificando gasto de @%s: %v", user, err)
		b.sendMessage(message.Chat.ID, "No pude clasificar ese gasto: "+err.Error())
		return
	}
	log.Printf("Gasto clasificado: $%.2f %s (%s) — confianza: %.0f%%", result.Amount, result.Description, result.Category, result.Confidence*100)

	if result.Installments >= 2 {
		b.handleInstallmentExpense(message, user, result)
		return
	}

	expense := &models.Expense{
		User:        user,
		Amount:      result.Amount,
		Description: result.Description,
		Category:    models.Category{Name: result.Category},
		RawMessage:  message.Text,
	}
	if result.Date != "" {
		expense.CreatedAt = result.Date + " 12:00:00"
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
	if result.Date != "" {
		sb.WriteString(fmt.Sprintf("📅 Fecha: %s\n", result.Date))
	}

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) handleInstallmentExpense(message *tgbotapi.Message, user string, result *agentModels.ClassificationResult) {
	expense := &models.Expense{
		User:        user,
		Description: result.Description,
		Category:    models.Category{Name: result.Category},
		RawMessage:  message.Text,
	}
	if result.Date != "" {
		expense.CreatedAt = result.Date + " 12:00:00"
	}

	installments, err := b.svc.CreateInstallments(expense, result.Amount, result.Installments)
	if err != nil {
		log.Printf("Error creando mensualidades para @%s: %v", user, err)
		b.sendMessage(message.Chat.ID, "No se pudieron registrar las mensualidades: "+err.Error())
		return
	}
	log.Printf("Mensualidades creadas: %d pagos de $%.2f para @%s", len(installments), installments[0].Amount, user)

	perMonth := installments[0].Amount
	firstDate := installments[0].CreatedAt[:7]
	lastDate := installments[len(installments)-1].CreatedAt[:7]

	var sb strings.Builder
	sb.WriteString("Compra a meses registrada:\n\n")
	sb.WriteString(fmt.Sprintf("💰 Total: $%.2f\n", result.Amount))
	sb.WriteString(fmt.Sprintf("📅 %d pagos de $%.2f\n", result.Installments, perMonth))
	sb.WriteString(fmt.Sprintf("📁 Categoría: %s\n", result.Category))
	sb.WriteString(fmt.Sprintf("📝 %s\n", result.Description))
	sb.WriteString(fmt.Sprintf("🗓️ Período: %s a %s\n", formatMonth(firstDate), formatMonth(lastDate)))
	sb.WriteString(fmt.Sprintf("🎯 Confianza: %.0f%%\n", result.Confidence*100))

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) handleResumen(message *tgbotapi.Message) {
	user := getUser(message)
	data, err := b.svc.GetDashboardData(user, "month", "", "", "")
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

	budgets, _ := b.budgetSvc.ListByUser(user)

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
		budgetMap := makeBudgetMap(budgets)
		for _, c := range data.ByCategory {
			line := fmt.Sprintf("  %s: $%.2f", c.Category, c.Total)
			if b, ok := budgetMap[c.Category]; ok {
				pct := (c.Total / b) * 100
				status := "✅"
				if pct >= 100 {
					status = "🔴"
				} else if pct >= 80 {
					status = "⚠️"
				}
				line += fmt.Sprintf(" / $%.0f (%.0f%%) %s", b, pct, status)
			}
			sb.WriteString(line + "\n")
		}
	}

	insightPrompt := buildInsightPrompt(data, budgets)
	insights, err := b.agent.GetInsights(context.Background(), insightPrompt)
	if err != nil {
		log.Printf("Error generando insights: %v", err)
	} else {
		sb.WriteString("\n💡 Insights:\n")
		sb.WriteString(insights)
	}

	b.sendMessage(message.Chat.ID, sb.String())
}

func (b *Bot) handleUltimos(message *tgbotapi.Message) {
	user := getUser(message)
	filter := models.ExpenseFilter{User: user, Period: "month"}
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

func (b *Bot) handlePresupuesto(message *tgbotapi.Message) {
	user := getUser(message)
	budgets, err := b.budgetSvc.ListByUser(user)
	if err != nil {
		b.sendMessage(message.Chat.ID, "Error obteniendo presupuesto: "+err.Error())
		return
	}

	var sb strings.Builder
	if len(budgets) == 0 {
		sb.WriteString("No tienes presupuesto configurado aún.\n")
		sb.WriteString("Usa el botón de abajo para configurar uno.")
	} else {
		sb.WriteString("Presupuesto mensual:\n\n")
		total := 0.0
		for _, bg := range budgets {
			sb.WriteString(fmt.Sprintf("  %s: $%.2f\n", bg.Category, bg.Amount))
			total += bg.Amount
		}
		sb.WriteString(fmt.Sprintf("\nTotal presupuestado: $%.2f", total))
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, sb.String())
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Editar presupuesto", "edit_budget"),
		),
	)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error enviando mensaje: %v", err)
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	ack := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(ack)

	data := callback.Data
	chatID := callback.Message.Chat.ID

	switch {
	case data == "edit_budget":
		b.sendCategoryKeyboard(chatID)

	case data == "budget_done":
		b.sendMessage(chatID, "Presupuesto actualizado.")

	case strings.HasPrefix(data, "budget:"):
		category := strings.TrimPrefix(data, "budget:")
		b.mu.Lock()
		b.pendingBudget[chatID] = category
		b.mu.Unlock()
		b.sendMessage(chatID, fmt.Sprintf("¿Cuánto asignas mensualmente a %s?\nEnvía el monto (ej: 1500):", category))
	}
}

func (b *Bot) sendCategoryKeyboard(chatID int64) {
	categories := service.ValidCategories()
	var rows [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < len(categories); i += 3 {
		end := i + 3
		if end > len(categories) {
			end = len(categories)
		}
		var row []tgbotapi.InlineKeyboardButton
		for _, cat := range categories[i:end] {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(cat, "budget:"+cat))
		}
		rows = append(rows, row)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Listo ✅", "budget_done"),
	))

	msg := tgbotapi.NewMessage(chatID, "Selecciona la categoría a configurar:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error enviando teclado de categorías: %v", err)
	}
}

func (b *Bot) handleBudgetAmount(message *tgbotapi.Message, category string) {
	b.mu.Lock()
	delete(b.pendingBudget, message.Chat.ID)
	b.mu.Unlock()

	amount, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || amount <= 0 {
		b.sendMessage(message.Chat.ID, "Monto inválido. Envía un número mayor a 0.")
		return
	}

	user := getUser(message)
	budget := &models.Budget{
		User:     user,
		Category: category,
		Amount:   amount,
	}

	if err := b.budgetSvc.Upsert(budget); err != nil {
		b.sendMessage(message.Chat.ID, "Error guardando presupuesto: "+err.Error())
		return
	}

	log.Printf("Presupuesto actualizado: @%s %s = $%.2f", user, category, amount)
	b.sendMessage(message.Chat.ID, fmt.Sprintf("Presupuesto de %s actualizado a $%.2f", category, amount))
	b.sendCategoryKeyboard(message.Chat.ID)
}

func (b *Bot) hasPendingBudget(chatID int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.pendingBudget[chatID]
	return ok
}

func (b *Bot) sendWeeklySummaries() {
	log.Println("Enviando resúmenes semanales...")

	chats, err := b.budgetSvc.ListChatIDs()
	if err != nil {
		log.Printf("Error obteniendo chat IDs: %v", err)
		return
	}

	for _, chat := range chats {
		data, err := b.svc.GetDashboardData(chat.User, "week", "", "", "")
		if err != nil {
			log.Printf("Error obteniendo datos semanales para @%s: %v", chat.User, err)
			continue
		}

		budgets, _ := b.budgetSvc.ListByUser(chat.User)

		var sb strings.Builder
		sb.WriteString("📅 Resumen semanal:\n\n")
		sb.WriteString(fmt.Sprintf("💰 Total de la semana: $%.2f\n", data.TotalAmount))
		sb.WriteString(fmt.Sprintf("📊 Gastos: %d\n", data.ExpenseCount))

		if len(data.ByCategory) > 0 {
			sb.WriteString("\nPor categoría:\n")
			budgetMap := makeBudgetMap(budgets)
			for _, c := range data.ByCategory {
				line := fmt.Sprintf("  %s: $%.2f", c.Category, c.Total)
				if b, ok := budgetMap[c.Category]; ok {
					pct := (c.Total / b) * 100
					status := "✅"
					if pct >= 100 {
						status = "🔴"
					} else if pct >= 80 {
						status = "⚠️"
					}
					line += fmt.Sprintf(" / $%.0f (%.0f%%) %s", b, pct, status)
				}
				sb.WriteString(line + "\n")
			}
		}

		insightPrompt := buildInsightPrompt(data, budgets)
		insights, err := b.agent.GetInsights(context.Background(), insightPrompt)
		if err != nil {
			log.Printf("Error generando insights semanales: %v", err)
		} else {
			sb.WriteString("\n💡 Insights:\n")
			sb.WriteString(insights)
		}

		b.sendMessage(chat.ChatID, sb.String())
		log.Printf("Resumen semanal enviado a @%s", chat.User)
	}
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

func getUser(message *tgbotapi.Message) string {
	if message.From.UserName != "" {
		return message.From.UserName
	}
	return message.From.FirstName
}

func makeBudgetMap(budgets []models.Budget) map[string]float64 {
	m := make(map[string]float64)
	for _, b := range budgets {
		m[b.Category] = b.Amount
	}
	return m
}

func buildInsightPrompt(data *models.DashboardData, budgets []models.Budget) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Total gastado: $%.2f\n", data.TotalAmount))
	sb.WriteString(fmt.Sprintf("Promedio diario: $%.2f\n", data.DailyAverage))
	sb.WriteString(fmt.Sprintf("Número de gastos: %d\n", data.ExpenseCount))

	if data.PrevTotal > 0 {
		diff := ((data.TotalAmount - data.PrevTotal) / data.PrevTotal) * 100
		sb.WriteString(fmt.Sprintf("vs. período anterior: %+.0f%%\n", diff))
	}

	budgetMap := makeBudgetMap(budgets)
	totalBudget := 0.0
	for _, b := range budgets {
		totalBudget += b.Amount
	}
	if totalBudget > 0 {
		sb.WriteString(fmt.Sprintf("Presupuesto total: $%.2f\n", totalBudget))
	}

	sb.WriteString("\nPor categoría (gastado / presupuesto):\n")
	for _, c := range data.ByCategory {
		line := fmt.Sprintf("- %s: $%.2f", c.Category, c.Total)
		if b, ok := budgetMap[c.Category]; ok {
			pct := (c.Total / b) * 100
			status := "OK"
			if pct >= 100 {
				status = "SOBRE PRESUPUESTO"
			}
			line += fmt.Sprintf(" / $%.0f (%.0f%%) %s", b, pct, status)
		} else {
			line += " (sin presupuesto)"
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

var monthNames = map[string]string{
	"01": "enero", "02": "febrero", "03": "marzo", "04": "abril",
	"05": "mayo", "06": "junio", "07": "julio", "08": "agosto",
	"09": "septiembre", "10": "octubre", "11": "noviembre", "12": "diciembre",
}

func formatMonth(ym string) string {
	parts := strings.Split(ym, "-")
	if len(parts) != 2 {
		return ym
	}
	name, ok := monthNames[parts[1]]
	if !ok {
		return ym
	}
	return name + " " + parts[0]
}

func nextWeekday(day time.Weekday, hour, minute int) time.Time {
	now := timeutil.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	daysUntil := int(day - now.Weekday())
	if daysUntil < 0 {
		daysUntil += 7
	}
	target = target.AddDate(0, 0, daysUntil)

	if !target.After(now) {
		target = target.AddDate(0, 0, 7)
	}

	return target
}
