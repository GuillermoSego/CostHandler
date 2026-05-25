package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/GuillermoSego/costhandler/agent/agent"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot tiene dos dependencias:
// - api: la conexión con Telegram (para recibir y enviar mensajes)
// - agent: el cerebro que clasifica gastos (Fase 2)
type Bot struct {
	api   *tgbotapi.BotAPI
	agent *agent.Agent
}

// NewBot crea la conexión con Telegram usando el token de BotFather.
// Si el token es inválido, Telegram rechaza y devuelve error.
func NewBot(token string, agent *agent.Agent) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("connecting to telegram: %w", err)
	}

	log.Printf("Bot conectado como: @%s", api.Self.UserName)

	return &Bot{
		api:   api,
		agent: agent,
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
	switch message.Command() {
	case "start":
		response = "¡Hola! Soy CostHandler, tu asistente de gastos.\n\n" +
			"Envíame un mensaje con tu gasto y yo lo clasifico.\n" +
			"Ejemplo: \"150 tacos al pastor\"\n\n" +
			"Comandos:\n" +
			"/resumen — resumen del mes\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/ayuda — ver esta lista"
	case "ayuda":
		response = "Comandos disponibles:\n" +
			"/start — bienvenida\n" +
			"/resumen — resumen del mes\n" +
			"/ultimos — últimos 5 gastos\n" +
			"/ayuda — ver esta lista"
	case "resumen":
		response = "Próximamente: resumen del mes"
	case "ultimos":
		response = "Próximamente: últimos gastos"
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

	// Llamamos al agent para clasificar el mensaje
	// context.Background() = context raíz, sin timeout por ahora
	result, err := b.agent.ProcessMessage(context.Background(), user, message.Text)
	if err != nil {
		b.sendMessage(message.Chat.ID, "No pude clasificar ese gasto: "+err.Error())
		return
	}

	// Formateamos la respuesta
	// strings.Builder es la forma eficiente de concatenar strings en Go
	var sb strings.Builder
	sb.WriteString("Gasto registrado:\n\n")
	sb.WriteString(fmt.Sprintf("💰 Monto: $%.2f\n", result.Amount))
	sb.WriteString(fmt.Sprintf("📁 Categoría: %s\n", result.Category))
	sb.WriteString(fmt.Sprintf("📝 Descripción: %s\n", result.Description))
	sb.WriteString(fmt.Sprintf("🎯 Confianza: %.0f%%\n", result.Confidence*100))

	b.sendMessage(message.Chat.ID, sb.String())
}

// sendMessage envía un mensaje de texto a un chat de Telegram.
// Lo separamos para no repetir el manejo de error en cada handler.
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error enviando mensaje: %v", err)
	}
}
