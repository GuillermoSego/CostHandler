package config

import (
	"os"
	"strings"
)

type Config struct {
	TelegramToken string
	OpenAIKey     string
	DBPath        string
	ServerPort    string
	BaseURL       string
	AllowedUsers  []string
}

func NewConfig() *Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	telegramToken := os.Getenv("TELEGRAM_TOKEN")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./expenses.db"
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	var allowedUsers []string
	if au := os.Getenv("ALLOWED_USERS"); au != "" {
		for _, u := range strings.Split(au, ",") {
			if trimmed := strings.TrimSpace(u); trimmed != "" {
				allowedUsers = append(allowedUsers, trimmed)
			}
		}
	}

	return &Config{
		TelegramToken: telegramToken,
		OpenAIKey:     openaiKey,
		DBPath:        dbPath,
		ServerPort:    port,
		BaseURL:       baseURL,
		AllowedUsers:  allowedUsers,
	}
}
