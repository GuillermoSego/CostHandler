# CostHandler

Rastreador de gastos personales mediante un bot de Telegram. Envias un mensaje en texto libre (ej. *"150 tacos al pastor"*), el bot lo clasifica automáticamente usando OpenAI y lo almacena en una base de datos SQLite. También expone una API JSON para consultar y administrar los gastos.

## Arquitectura

El proyecto se compone de tres módulos Go gestionados con un workspace (`go.work`):

```
CostHandler_bot  ──→  CostHandler_agent  ──→  CostHandler_mcp
   (Telegram)            (OpenAI)              (SQLite + API)
```

| Módulo | Responsabilidad |
|--------|----------------|
| **CostHandler_mcp** | Capa de datos: SQLite, repositorio, servicio de validación y handlers HTTP para la API JSON |
| **CostHandler_agent** | Capa de clasificación: cliente de OpenAI (`gpt-4o-mini`) que parsea texto libre a gasto estructurado |
| **CostHandler_bot** | Bot de Telegram: recibe mensajes, delega al agente y muestra resultados. Punto de entrada de la aplicación |

## Estructura del proyecto

```
CostHandler/
├── go.work
├── Dockerfile
├── .env.example
│
├── CostHandler_mcp/
│   ├── config/config.go             # Carga de variables de entorno
│   ├── models/expenses.go           # Modelos: Expense, Category
│   ├── database/sqlite.go           # Inicialización de SQLite y esquema
│   ├── repository/expense_repository.go  # Patrón repositorio
│   ├── service/expense_service.go   # Lógica de negocio y validación
│   ├── service/expense_service_test.go   # Tests unitarios
│   ├── handler/expense_handler.go   # Handlers HTTP
│   └── Makefile
│
├── CostHandler_agent/
│   ├── models/classification.go     # ClassificationResult
│   ├── openai/client.go             # Cliente HTTP para OpenAI
│   ├── agent/agent.go               # Orquestación y validación de confianza
│   └── Makefile
│
└── CostHandler_bot/
    ├── cmd/main.go                  # Entry point, inyección de dependencias
    └── bot/bot.go                   # Long polling de Telegram
```

## Requisitos previos

- **Go 1.22+**
- Una cuenta de [OpenAI](https://platform.openai.com/) con API key
- Un bot de Telegram creado con [@BotFather](https://t.me/BotFather)

## Configuración

Copia el archivo de ejemplo y completa con tus valores:

```bash
cp .env.example .env
```

Variables requeridas:

| Variable | Descripción | Valor por defecto |
|----------|-------------|-------------------|
| `TELEGRAM_TOKEN` | Token del bot de Telegram (de @BotFather) | — |
| `OPENAI_API_KEY` | API key de OpenAI | — |
| `DB_PATH` | Ruta del archivo SQLite | `./expenses.db` |
| `SERVER_PORT` | Puerto del servidor HTTP | `8080` |

## Cómo correr

### Ejecución local

```bash
# Cargar variables de entorno
source .env

# Ejecutar la aplicación
cd CostHandler_bot && go run ./cmd/main.go
```

Esto levanta dos procesos concurrentes:
- El **bot de Telegram** escuchando mensajes vía long polling
- El **servidor HTTP** en el puerto configurado

### Compilar binario

```bash
source .env
cd CostHandler_bot && go build -o costhandler ./cmd/main.go
./costhandler
```

### Con Docker

```bash
docker build -t costhandler .

docker run -d \
  --name costhandler \
  -e TELEGRAM_TOKEN=tu-token \
  -e OPENAI_API_KEY=tu-key \
  -e DB_PATH=/data/expenses.db \
  -e SERVER_PORT=8080 \
  -p 8080:8080 \
  -v costhandler-data:/data \
  costhandler
```

### Cross-compilation para ARM64 (servidores Oracle Cloud, Raspberry Pi, etc.)

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o costhandler ./cmd/main.go
```

## Tests

```bash
# Todos los tests del módulo de datos
cd CostHandler_mcp && go test ./... -v

# Un test específico
cd CostHandler_mcp && go test ./service -run TestCreate -v

# Via Makefile
cd CostHandler_mcp && make test
cd CostHandler_agent && make test
```

Los tests usan bases de datos SQLite en memoria (`:memory:`) y el patrón table-driven.

## API HTTP

El servidor expone los siguientes endpoints:

### Listar gastos

```
GET /api/expenses
```

### Crear gasto

```
POST /api/expenses
Content-Type: application/json

{
  "user": "guillermo",
  "amount": 150.0,
  "description": "Tacos al pastor",
  "category": { "name": "restaurantes" },
  "raw_message": "150 tacos al pastor"
}
```

### Eliminar gasto

```
DELETE /api/expenses/{id}
```

## Comandos del bot

| Comando | Descripción |
|---------|-------------|
| `/start` | Mensaje de bienvenida e instrucciones |
| `/ayuda` | Lista de comandos disponibles |
| `/resumen` | Resumen mensual (pendiente) |
| `/ultimos` | Últimos 5 gastos (pendiente) |

Para registrar un gasto simplemente envía un mensaje de texto libre, por ejemplo:

> 250 uber al aeropuerto

El bot responde con la clasificación, monto, descripción y nivel de confianza.

## Categorías válidas

`supermercado` · `restaurantes` · `vivienda` · `servicios` · `transporte` · `salud` · `familia` · `suscripciones` · `entretenimiento` · `compras` · `ahorro` · `otros`

## Esquema de base de datos

```sql
CREATE TABLE IF NOT EXISTS expenses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user        TEXT NOT NULL,
    amount      REAL NOT NULL,
    description TEXT NOT NULL,
    category    TEXT NOT NULL,
    raw_message TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Stack tecnológico

- **Go** con workspace multi-módulo
- **SQLite** vía `modernc.org/sqlite` (driver puro en Go, sin CGO)
- **OpenAI API** (`gpt-4o-mini`) para clasificación de gastos
- **Telegram Bot API** vía `go-telegram-bot-api/v5`
