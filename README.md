# CostHandler

Rastreador de gastos personales impulsado por inteligencia artificial. Un bot de Telegram recibe mensajes en texto libre (ej. *"150 tacos al pastor"*), los clasifica automáticamente usando **OpenAI GPT-4o-mini**, almacena el resultado en **SQLite** y lo expone a través de una **API REST JSON** y un **dashboard web interactivo**. Soporta presupuestos mensuales por categoría, compras a meses sin intereses, insights financieros generados por IA y resúmenes semanales automáticos.

---

## Stack tecnológico

| Capa | Tecnología | Detalle |
|------|-----------|---------|
| **Lenguaje** | Go 1.22+ | Workspace multi-módulo (`go.work`) con 3 módulos independientes |
| **Base de datos** | SQLite | Driver puro en Go (`modernc.org/sqlite`) — sin dependencia de CGO |
| **IA / NLP** | OpenAI API | Modelo `gpt-4o-mini` para clasificación de gastos e insights financieros |
| **Bot** | Telegram Bot API | Long polling con `go-telegram-bot-api/v5`, inline keyboards y callbacks |
| **HTTP** | `net/http` (stdlib) | Routing con `http.ServeMux` de Go 1.22 (pattern matching nativo con verbos) |
| **Frontend** | HTML/CSS/JS embebido | Templates Go (`html/template`) + assets estáticos via `embed.FS` |
| **Tipografía** | PP Object Sans | Fuentes custom embebidas en el binario (woff2) |
| **Contenedores** | Docker | Multi-stage build con Alpine — imagen final ~15 MB |
| **Zona horaria** | `timeutil` | Paquete propio que centraliza `time.Now()` respetando `TZ` |

## Arquitectura

El proyecto sigue una arquitectura en capas con inyección de dependencias manual y el patrón **Repository**:

```
CostHandler_bot  ──→  CostHandler_agent  ──→  CostHandler_mcp
   (Telegram)            (OpenAI)              (SQLite + API + Dashboard)
```

### Flujo de datos

```
Mensaje Telegram ──→ Bot ──→ Agent (OpenAI) ──→ ClassificationResult
                                                        │
                                                        ▼
                                              ExpenseService (validación)
                                                        │
                                                        ▼
                                              ExpenseRepository (SQLite)
                                                        │
                                                        ▼
                                              API JSON + Dashboard Web
```

### Módulos

| Módulo | Responsabilidad |
|--------|----------------|
| **CostHandler_mcp** | Capa de datos y presentación: SQLite, patrón repositorio, servicio de validación/negocio, handlers HTTP (API JSON + dashboard), modelos, paquete `timeutil`, assets web embebidos |
| **CostHandler_agent** | Capa de inteligencia artificial: cliente HTTP directo a la API de OpenAI, clasificación de texto libre a gasto estructurado, generación de insights financieros |
| **CostHandler_bot** | Capa de interacción: bot de Telegram con long polling, inline keyboards, gestión de presupuestos conversacional, scheduler de resúmenes semanales, control de acceso por usuario |

### Patrones y métodos utilizados

- **Repository Pattern** — Interfaces `ExpenseRepository` y `BudgetRepository` con implementaciones SQLite, desacoplan la lógica de negocio del storage
- **Dependency Injection manual** — Cada capa recibe sus dependencias por constructor (`NewExpenseService(repo)`, `NewBot(token, agent, svc, budgetSvc, ...)`)
- **Table-driven tests** — Tests unitarios con SQLite `:memory:` y múltiples casos de prueba por tabla
- **Embed filesystem** — Templates HTML y assets estáticos compilados dentro del binario con `//go:embed`
- **Multi-stage Docker build** — Stage de compilación con `golang:1.22-alpine`, imagen final con `alpine:3.19` (solo binario + tzdata)
- **Goroutines concurrentes** — Bot de Telegram y servidor HTTP corren en paralelo via `go`
- **Migraciones automáticas** — `RunMigrations()` detecta el esquema actual con `PRAGMA table_info` y agrega columnas faltantes
- **Transacciones batch** — Inserción de mensualidades en una sola transacción SQL con `Prepare`/`Exec`/`Commit`

## Estructura del proyecto

```
CostHandler/
├── go.work                              # Workspace multi-módulo
├── Dockerfile                           # Multi-stage build
├── .env.example
│
├── CostHandler_mcp/
│   ├── config/config.go                 # Carga de variables de entorno
│   ├── models/
│   │   ├── expenses.go                  # Expense, Category (con soporte mensualidades)
│   │   ├── budget.go                    # Budget, BudgetComparison, UserChat
│   │   └── dashboard.go                 # DashboardData, filtros, sumarios
│   ├── database/sqlite.go              # Inicialización de SQLite, tablas y migraciones
│   ├── repository/
│   │   ├── expense_repository.go        # CRUD + queries analíticas (SumByCategory, SumByDay, SumByMonth)
│   │   └── budget_repository.go         # CRUD de presupuestos + registro de chat IDs
│   ├── service/
│   │   ├── expense_service.go           # Lógica de negocio, mensualidades, datos de dashboard
│   │   ├── expense_service_test.go      # Tests unitarios con table-driven pattern
│   │   └── budget_service.go            # Gestión de presupuestos y comparación vs. gastos reales
│   ├── handler/
│   │   ├── expense_handler.go           # API REST JSON + endpoints de dashboard y presupuestos
│   │   └── dashboard_handler.go         # Servir HTML del dashboard + archivos estáticos
│   ├── timeutil/timeutil.go             # Zona horaria centralizada (configurable via TZ)
│   ├── web/
│   │   ├── embed.go                     # go:embed para templates y static
│   │   ├── templates/dashboard.html     # Dashboard interactivo
│   │   └── static/
│   │       ├── dashboard.css            # Estilos del dashboard
│   │       ├── dashboard.js             # Lógica frontend (charts, filtros, CRUD)
│   │       └── fonts/                   # PP Object Sans (woff2)
│   ├── cmd/migrate_tz/main.go           # Herramienta de migración de zona horaria
│   └── Makefile
│
├── CostHandler_agent/
│   ├── models/classification.go         # ClassificationResult (incluye installments)
│   ├── openai/client.go                 # Cliente HTTP para OpenAI (Classify + GenerateInsights)
│   ├── agent/agent.go                   # Orquestación, validación de confianza e insights
│   └── Makefile
│
└── CostHandler_bot/
    ├── cmd/main.go                      # Entry point, inyección de dependencias, arranque concurrente
    └── bot/bot.go                       # Long polling, comandos, inline keyboards, scheduler semanal
```

## Funcionalidades

### Registro de gastos por texto libre
Envía cualquier mensaje al bot (ej. *"250 uber al aeropuerto"*) y la IA extrae automáticamente monto, categoría, descripción y nivel de confianza. Si la confianza es menor al 50%, el gasto se rechaza.

### Compras a meses sin intereses
Mensajes como *"12000 laptop a 6 meses"* generan automáticamente 6 registros con montos prorrateados, distribuidos uno por mes futuro. Se usa una transacción SQL para garantizar atomicidad. Los gastos comparten un `installment_group` UUID para agruparlos.

### Presupuestos mensuales por categoría
Desde el comando `/presupuesto`, el bot muestra un **inline keyboard** con todas las categorías para asignar montos. Usa un sistema conversacional de dos pasos: seleccionar categoría → enviar monto. Los presupuestos se almacenan con `UPSERT` (INSERT OR REPLACE) para simplificar actualizaciones.

### Dashboard web interactivo
Accesible en `/dashboard`, muestra gastos por categoría, tendencia diaria, histórico mensual, y comparación presupuesto vs. gasto real. Filtrable por período (semana/mes/año), categoría y usuario. Los templates HTML y assets CSS/JS están embebidos en el binario con `//go:embed`.

### Insights financieros con IA
El comando `/resumen` muestra el resumen mensual y envía los datos a OpenAI para generar **insights accionables**: categorías sobre presupuesto, tendencias preocupantes, oportunidades de ahorro. El prompt incluye datos reales de gasto vs. presupuesto.

### Resúmenes semanales automáticos
Cada lunes a las 9:00 AM (hora local), el bot envía automáticamente un resumen semanal a todos los usuarios registrados. Incluye desglose por categoría, comparación con presupuesto e insights de IA. Usa un scheduler basado en goroutines con `nextWeekday()`.

### Control de acceso
Variable `ALLOWED_USERS` limita qué usuarios de Telegram pueden interactuar con el bot. Si no se configura, todos los usuarios tienen acceso.

### API REST completa
API JSON para integrar con otros sistemas o el dashboard web.

## Comandos del bot

| Comando | Descripción |
|---------|-------------|
| `/start` | Mensaje de bienvenida e instrucciones |
| `/ayuda` | Lista de comandos disponibles |
| `/resumen` | Resumen mensual con desglose por categoría + insights de IA |
| `/ultimos` | Últimos 5 gastos del mes |
| `/presupuesto` | Ver y editar presupuesto mensual con inline keyboard |
| `/dashboard` | Link al dashboard web |

Para registrar un gasto simplemente envía un mensaje de texto libre:

> 250 uber al aeropuerto

> 12000 laptop a 6 meses sin intereses

## API HTTP

### Gastos

```
GET    /api/expenses              # Listar gastos (?period=month&category=restaurantes&user=nombre)
POST   /api/expenses              # Crear gasto
PUT    /api/expenses/{id}         # Actualizar gasto
DELETE /api/expenses/{id}         # Eliminar gasto
```

### Dashboard

```
GET    /api/dashboard/summary     # Datos agregados (?period=month&user=nombre&category=restaurantes)
```

Respuesta:
```json
{
  "total_amount": 15420.50,
  "daily_average": 514.02,
  "top_category": "supermercado",
  "top_category_amount": 5200.00,
  "prev_total": 13800.00,
  "by_category": [{"category": "supermercado", "total": 5200.00, "count": 12}],
  "by_day": [{"date": "2026-05-01", "total": 350.00}],
  "by_month": [{"month": "2026-05", "total": 15420.50}],
  "expense_count": 45,
  "budget_comparison": [{"category": "supermercado", "budgeted": 6000, "spent": 5200, "remaining": 800, "percentage": 86.67}],
  "total_budgeted": 25000.00
}
```

### Presupuestos

```
GET    /api/budgets               # Listar presupuestos (?user=nombre)
POST   /api/budgets               # Crear/actualizar presupuesto (upsert)
```

### Usuarios

```
GET    /api/users                 # Listar usuarios distintos
```

### Dashboard web

```
GET    /dashboard                 # Página HTML del dashboard
GET    /static/*                  # Assets estáticos (CSS, JS, fuentes)
```

## Categorías válidas

`supermercado` · `restaurantes` · `vivienda` · `servicios` · `transporte` · `salud` · `familia` · `suscripciones` · `entretenimiento` · `compras` · `ahorro` · `otros`

## Esquema de base de datos

```sql
CREATE TABLE expenses (
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
);

CREATE TABLE budgets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user       TEXT NOT NULL,
    category   TEXT NOT NULL,
    amount     REAL NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user, category)
);

CREATE TABLE user_chats (
    user    TEXT PRIMARY KEY,
    chat_id INTEGER NOT NULL
);
```

## Requisitos previos

- **Go 1.22+**
- Una cuenta de [OpenAI](https://platform.openai.com/) con API key
- Un bot de Telegram creado con [@BotFather](https://t.me/BotFather)

## Configuración

```bash
cp .env.example .env
```

| Variable | Descripción | Valor por defecto |
|----------|-------------|-------------------|
| `TELEGRAM_TOKEN` | Token del bot de Telegram (de @BotFather) | — |
| `OPENAI_API_KEY` | API key de OpenAI | — |
| `DB_PATH` | Ruta del archivo SQLite | `./expenses.db` |
| `SERVER_PORT` | Puerto del servidor HTTP | `8080` |
| `BASE_URL` | URL base para links del dashboard | `http://localhost:{port}` |
| `TZ` | Zona horaria para timestamps | `America/Mexico_City` |
| `ALLOWED_USERS` | Usernames de Telegram autorizados (separados por coma) | todos permitidos |

## Cómo correr

### Ejecución local

```bash
source .env
cd CostHandler_bot && go run ./cmd/main.go
```

Esto levanta tres procesos concurrentes:
- El **bot de Telegram** escuchando mensajes vía long polling
- El **servidor HTTP** con API JSON + dashboard en el puerto configurado
- El **scheduler semanal** que envía resúmenes automáticos los lunes a las 9 AM

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
  -e TZ=America/Mexico_City \
  -e ALLOWED_USERS=tu_usuario \
  -p 8080:8080 \
  -v costhandler-data:/data \
  costhandler
```

### Cross-compilation para ARM64 (Oracle Cloud, Raspberry Pi)

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

Los tests usan bases de datos SQLite en memoria (`:memory:`) y el patrón **table-driven** para cubrir múltiples escenarios por test.
