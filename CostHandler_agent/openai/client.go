package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GuillermoSego/costhandler/agent/models"
)

// Client guarda la configuración para hablar con OpenAI.
// No hace ninguna llamada HTTP hasta que alguien llame a Classify.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient solo PREPARA el cliente — guarda la API key y configura un timeout.
// Es como guardar el número de teléfono; todavía no llamamos.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Si OpenAI no responde en 30s, cancelamos
		},
		baseURL: "https://api.openai.com/v1/chat/completions",
	}
}

// --- Structs para el JSON que manda y recibe la API de OpenAI ---
// Estos son "internos" — solo los usa este paquete para serializar/deserializar.

// chatRequest es lo que le MANDAMOS a OpenAI
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage es cada mensaje en la conversación (system, user, assistant)
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse es lo que OpenAI nos DEVUELVE
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func buildSystemPrompt() string {
	now := time.Now()
	currentYear := now.Format("2006")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	dayBefore := now.AddDate(0, 0, -2).Format("2006-01-02")

	return fmt.Sprintf(`Eres un asistente que clasifica gastos personales.
El usuario te va a enviar un mensaje describiendo un gasto.
Tu trabajo es extraer: el monto, la categoría, una descripción corta, la fecha si se menciona, y si es a meses sin intereses.

Categorías válidas: supermercado, restaurantes, vivienda, servicios, transporte,
salud, familia, suscripciones, entretenimiento, compras, ahorro, otros.

Si el mensaje menciona "meses sin intereses", "a X meses", "X mensualidades", o similar:
- "amount" debe ser el MONTO TOTAL de la compra
- "installments" debe ser el número de meses (ej: 6, 12, 18)

Si NO menciona meses sin intereses, usa "installments": 0.

Para la fecha:
- Si el mensaje menciona una fecha (ej: "el 28 de abril", "el 15 de marzo"), extráela en formato "YYYY-MM-DD".
- Si solo menciona día y mes sin año, asume el año actual: %s.
- Si dice "ayer", usa la fecha: %s.
- Si dice "antier" o "anteayer", usa la fecha: %s.
- Si NO menciona ninguna fecha, usa "date": "".

Responde ÚNICAMENTE con JSON válido, sin markdown, sin texto extra:
{"amount": 5000.00, "category": "compras", "description": "Audífonos", "confidence": 0.95, "installments": 6, "date": ""}

Si no puedes determinar el monto, usa 0 y confidence bajo.
Si no puedes determinar la categoría, usa "otros" y confidence bajo.`, currentYear, yesterday, dayBefore)
}

// Classify envía un mensaje a OpenAI y devuelve el gasto clasificado.
// AQUÍ es donde se hace el HTTP POST — no en el constructor.
func (c *Client) Classify(ctx context.Context, message string) (*models.ClassificationResult, error) {
	// 1. Armamos el body del request (lo que le mandamos a OpenAI)
	reqBody := chatRequest{
		Model: "gpt-4o-mini",
		Messages: []chatMessage{
			{Role: "system", Content: buildSystemPrompt()},
			{Role: "user", Content: message},
		},
	}

	// 2. Convertimos el struct a JSON (como JSON.stringify en JS)
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// 3. Creamos el HTTP request con context (para poder cancelarlo)
	// bytes.NewReader convierte el []byte a algo que http.NewRequest puede leer
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// 4. Headers — OpenAI necesita el token y saber que mandamos JSON
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 5. Enviamos el request (AQUÍ es donde sale el HTTP POST al internet)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling openai: %w", err)
	}
	// defer cierra el body cuando la función termine — libera la conexión HTTP
	defer resp.Body.Close()

	// 6. Leemos todo el body de la respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// 7. Si OpenAI devolvió un error (401, 429, 500, etc.)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	// 8. Parseamos la respuesta de OpenAI (el wrapper con choices, etc.)
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing openai response: %w", err)
	}

	// 9. Verificamos que haya al menos una respuesta
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	// 10. El contenido del mensaje es un string JSON — lo parseamos a nuestro struct
	var result models.ClassificationResult
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("parsing classification result: %w", err)
	}

	return &result, nil
}

const insightsSystemPrompt = `Eres un asesor financiero personal. Analiza los gastos del usuario comparados con su presupuesto mensual.
Da máximo 5 insights concretos y accionables. Usa emojis. Sé directo y específico.
Enfócate en: categorías sobre presupuesto, tendencias preocupantes, oportunidades de ahorro, y patrones positivos.
Responde en español. IMPORTANTE: No uses markdown. No uses asteriscos (**), guiones bajos (__), ni backticks. Solo texto plano con emojis.`

func (c *Client) GenerateInsights(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: "gpt-4o-mini",
		Messages: []chatMessage{
			{Role: "system", Content: insightsSystemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling openai: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parsing openai response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	result := chatResp.Choices[0].Message.Content
	result = strings.ReplaceAll(result, "**", "")
	result = strings.ReplaceAll(result, "__", "")
	result = strings.ReplaceAll(result, "```", "")
	return result, nil
}
