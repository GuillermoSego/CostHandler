# CostHandler — Plan de Implementación

> Agente personal de gastos: Telegram Bot + OpenAI + SQLite + Dashboard
> Objetivo: Aprender Go construyendo algo funcional

---

## Stack definido

| Componente | Tecnología | Costo |
|---|---|---|
| Chat | Telegram (go-telegram-bot-api) | $0 |
| AI | OpenAI GPT-4o-mini | ~$0.15/1M tokens |
| DB | SQLite (modernc.org/sqlite) | $0 |
| Dashboard | HTML + Chart.js (servido desde Go) | $0 |
| Hosting | Local / VPS barato | $0-5/mes |

---

## Fase 0: Fundamentos del proyecto

**Objetivo de aprendizaje:** Go modules, estructura de proyecto, tipos básicos.

### 0.1 — Inicializar Go modules
- Crear `go.mod` para cada módulo: `costhandler_mcp`, `costhandler_agent`, `costhandler_bot`
- Definir un módulo raíz `costhandler` con un `go.work` workspace
- **Aprenderás:** Go modules, workspaces, versionado semántico

### 0.2 — Definir modelos compartidos
- Crear paquete `internal/models` con los tipos base:
  - `Expense` (ID, Amount, Description, Category, Date, RawMessage)
  - `Category` (ID, Name, Icon)
  - `ClassificationResult` (Category, Confidence, ParsedAmount)
- **Aprenderás:** Structs, tipos, tags JSON, paquetes internos

### 0.3 — Configuración y variables de entorno
- Crear paquete `internal/config` para leer env vars
  - `TELEGRAM_TOKEN`, `OPENAI_API_KEY`, `DB_PATH`, `SERVER_PORT`
- **Aprenderás:** `os.Getenv`, manejo de configuración, paquete `flag`

---

## Fase 1: CostHandler_mcp (Capa de datos)

**Objetivo de aprendizaje:** Interfaces, SQL en Go, repository pattern, HTTP server.

### 1.1 — Schema SQLite
- Crear tabla `expenses`:
  ```sql
  CREATE TABLE expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    amount REAL NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    raw_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  ```
- Crear tabla `categories` con categorías predefinidas
- **Aprenderás:** `database/sql`, drivers SQLite en Go, migrations

### 1.2 — Repository de gastos
- Definir interfaz `ExpenseRepository`:
  ```go
  type ExpenseRepository interface {
    Create(ctx context.Context, expense *Expense) error
    GetByID(ctx context.Context, id int64) (*Expense, error)
    List(ctx context.Context, filter ExpenseFilter) ([]Expense, error)
    GetSummary(ctx context.Context, period string) (*Summary, error)
    Delete(ctx context.Context, id int64) error
  }
  ```
- Implementar `SQLiteExpenseRepository`
- **Aprenderás:** Interfaces, context.Context, error handling, query building

### 1.3 — Servicio de datos (business logic)
- Crear `ExpenseService` que use el repository
- Validaciones: monto > 0, categoría válida, fecha no futura
- Agregaciones: gasto por categoría, por mes, totales
- **Aprenderás:** Composición sobre herencia, validación, service layer

### 1.4 — HTTP API (endpoints JSON)
- `POST /api/expenses` — crear gasto
- `GET /api/expenses` — listar con filtros (fecha, categoría)
- `GET /api/expenses/summary` — resumen por período
- `DELETE /api/expenses/:id` — eliminar gasto
- `GET /api/categories` — listar categorías
- **Aprenderás:** `net/http`, JSON marshaling, middleware, routing

### 1.5 — Tests del MCP
- Tests unitarios del repository con SQLite in-memory
- Tests de los endpoints HTTP con `httptest`
- **Aprenderás:** `testing`, table-driven tests, test fixtures

---

## Fase 2: CostHandler_agent (Cerebro de clasificación)

**Objetivo de aprendizaje:** HTTP clients, JSON, prompts, goroutines.

### 2.1 — Cliente OpenAI
- Crear cliente HTTP para la API de OpenAI (chat completions)
- Manejar rate limits, timeouts, retries
- **Aprenderás:** `net/http` client, JSON encode/decode, error wrapping

### 2.2 — Prompt de clasificación
- Diseñar prompt que reciba texto libre y devuelva:
  ```json
  {
    "amount": 150.00,
    "category": "transporte",
    "description": "Uber al trabajo",
    "confidence": 0.95
  }
  ```
- Categorías predefinidas: alimentación, transporte, entretenimiento, salud, hogar, servicios, educación, otros
- **Aprenderás:** Prompt engineering, structured output, JSON parsing

### 2.3 — Orquestador del agente
- Crear `Agent` struct que coordine:
  1. Recibir mensaje crudo del bot
  2. Llamar a OpenAI para clasificar
  3. Si confidence < 0.7, pedir confirmación al usuario
  4. Guardar en MCP via HTTP o llamada directa
  5. Devolver confirmación formateada
- **Aprenderás:** Composición de servicios, state machine, channels

### 2.4 — Tests del agent
- Mock del cliente OpenAI para tests
- Tests de clasificación con mensajes reales
- **Aprenderás:** Interfaces para mocking, dependency injection

---

## Fase 3: CostHandler_bot (Interfaz Telegram)

**Objetivo de aprendizaje:** Goroutines, channels, external APIs, command handling.

### 3.1 — Bot básico de Telegram
- Registrar bot con @BotFather
- Configurar long polling para desarrollo
- Responder a `/start` con mensaje de bienvenida
- **Aprenderás:** go-telegram-bot-api, goroutines, event loop

### 3.2 — Handlers de comandos
- `/start` — bienvenida + instrucciones
- `/resumen` — resumen del mes actual
- `/categorias` — listar categorías disponibles
- `/ultimos` — últimos 5 gastos
- `/eliminar <id>` — eliminar un gasto
- `/ayuda` — lista de comandos
- **Aprenderás:** Command pattern, switch/routing, string formatting

### 3.3 — Manejo de mensajes libres (gastos)
- Cualquier mensaje que no sea comando → enviarlo al agent
- Recibir clasificación → mostrar con inline keyboard para confirmar/corregir
- Botones: ✅ Confirmar | ✏️ Cambiar categoría | ❌ Cancelar
- **Aprenderás:** Inline keyboards, callback queries, conversational state

### 3.4 — Flujo de confirmación
- Si el usuario confirma → guardar via agent → MCP
- Si cambia categoría → mostrar lista de categorías como botones
- Si cancela → descartar y confirmar
- **Aprenderás:** State management, goroutine patterns

### 3.5 — Tests del bot
- Tests de handlers con mocks del agent
- Tests de parsing de comandos
- **Aprenderás:** Testing con dependencias externas

---

## Fase 4: Integración

**Objetivo de aprendizaje:** Wiring, dependency injection manual, graceful shutdown.

### 4.1 — Main: conectar todo
- Crear `cmd/costhandler/main.go` que:
  1. Cargue configuración
  2. Inicialice SQLite + repository
  3. Inicialice el agent con OpenAI client
  4. Inicie el bot de Telegram
  5. Inicie el HTTP server (dashboard + API)
- **Aprenderás:** Dependency injection manual, composition root

### 4.2 — Graceful shutdown
- Escuchar signals (SIGINT, SIGTERM)
- Cerrar bot, HTTP server, DB connection en orden
- **Aprenderás:** `os/signal`, `context.WithCancel`, defer, goroutine coordination

### 4.3 — Test de integración end-to-end
- Simular: mensaje → clasificación → almacenamiento → consulta
- Verificar flujo completo con SQLite in-memory
- **Aprenderás:** Integration testing, test helpers

---

## Fase 5: Dashboard

**Objetivo de aprendizaje:** Templates, static files, frontend básico.

### 5.1 — HTML template base
- Página principal con layout responsive
- Header con título + período seleccionado
- **Aprenderás:** `html/template`, `embed` para static files

### 5.2 — Gráficas con Chart.js
- Donut chart: distribución por categoría
- Bar chart: gastos por día/semana
- Line chart: tendencia mensual
- **Aprenderás:** Servir static files, template data injection

### 5.3 — Tarjetas de resumen
- Total del mes
- Categoría top
- Promedio diario
- Comparación vs mes anterior
- **Aprenderás:** Agregaciones SQL, formateo de moneda

### 5.4 — Filtros interactivos
- Selector de período (semana/mes/año)
- Filtro por categoría
- Fetch asíncrono a la API JSON
- **Aprenderás:** API design, query parameters

---

## Fase 6: Polish y deployment

### 6.1 — Logging estructurado
- Usar `log/slog` (estándar en Go 1.21+)
- Logs con contexto: user_id, expense_id, latency
- **Aprenderás:** slog, structured logging

### 6.2 — Manejo de errores robusto
- Error types custom para cada capa
- Wrapping con `fmt.Errorf("...: %w", err)`
- **Aprenderás:** Error wrapping, sentinel errors, errors.Is/As

### 6.3 — Dockerfile
- Multi-stage build (build + runtime)
- SQLite file en volume
- **Aprenderás:** Docker con Go, scratch/alpine images

### 6.4 — README y documentación
- Setup instructions
- Variables de entorno
- Cómo crear el bot de Telegram

---

## Orden de ejecución sugerido

```
Fase 0 (setup)          ██░░░░░░░░░░░░░░  ~1 día
  ↓
Fase 1 (MCP/datos)      ████████░░░░░░░░  ~3 días
  ↓
Fase 2 (agent/AI)       ██████░░░░░░░░░░  ~2 días
  ↓
Fase 3 (bot/Telegram)   ██████░░░░░░░░░░  ~2 días
  ↓
Fase 4 (integración)    ████░░░░░░░░░░░░  ~1 día
  ↓
Fase 5 (dashboard)      ██████░░░░░░░░░░  ~2 días
  ↓
Fase 6 (polish)         ████░░░░░░░░░░░░  ~1 día
                                    Total: ~12 días
```

---

## Categorías de gastos predefinidas

| Categoría | Emoji | Ejemplos |
|---|---|---|
| Alimentación | 🍔 | Restaurantes, super, snacks |
| Transporte | 🚗 | Uber, gasolina, metro, taxi |
| Entretenimiento | 🎬 | Cine, streaming, juegos, salidas |
| Salud | 💊 | Farmacia, doctor, gym |
| Hogar | 🏠 | Renta, luz, agua, gas, internet |
| Servicios | 📱 | Celular, suscripciones, seguros |
| Educación | 📚 | Cursos, libros, materiales |
| Ropa | 👕 | Ropa, zapatos, accesorios |
| Otros | 📦 | Todo lo que no encaje arriba |

---

## Status tracking

| Fase | Status | Inicio | Fin |
|---|---|---|---|
| Fase 0: Setup | ⬜ Pendiente | - | - |
| Fase 1: MCP | ⬜ Pendiente | - | - |
| Fase 2: Agent | ⬜ Pendiente | - | - |
| Fase 3: Bot | ⬜ Pendiente | - | - |
| Fase 4: Integración | ⬜ Pendiente | - | - |
| Fase 5: Dashboard | ⬜ Pendiente | - | - |
| Fase 6: Polish | ⬜ Pendiente | - | - |
