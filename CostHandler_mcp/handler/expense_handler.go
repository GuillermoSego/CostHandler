package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/service"
)

type ExpenseHandler struct {
	serv *service.ExpenseService
}

func NewExpenseHandler(serv *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{serv: serv}
}

// HandleList — GET /api/expenses
// Este es el más sencillo: pide la lista al service y la devuelve como JSON.
func (h *ExpenseHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	// 1. Llamamos al service para obtener los gastos
	expenses, err := h.serv.List()
	if err != nil {
		// http.Error escribe un mensaje de error con el status code que le pases.
		// http.StatusInternalServerError = 500
		http.Error(w, "error listing expenses", http.StatusInternalServerError)
		return // IMPORTANTE: siempre return después de Error, si no sigue ejecutando
	}

	// 2. Decimos que la respuesta es JSON (si no, el browser/cliente no sabe qué formato es)
	w.Header().Set("Content-Type", "application/json")

	// 3. Convertimos el slice de expenses a JSON y lo escribimos en w (la respuesta)
	// json.NewEncoder(w) crea un encoder que escribe directo al ResponseWriter
	// .Encode(expenses) convierte el slice a JSON — usa los tags `json:"..."` de tus structs
	json.NewEncoder(w).Encode(expenses)
}

// HandleCreate — POST /api/expenses
// Tu turno: decodifica el JSON del body, llama a service.Create, responde 201 o error.
// Pistas:
//   - json.NewDecoder(r.Body).Decode(&expense) para leer el JSON del request
//   - w.WriteHeader(http.StatusCreated) para responder 201
func (h *ExpenseHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var expense models.Expense

	// Leemos el JSON del body y lo metemos en el struct expense.
	// Si el JSON está mal formado o no coincide con el struct, Decode devuelve error.
	if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return // Sin esto, la función sigue y trata de crear un expense vacío/roto
	}

	// Llamamos al service que valida y guarda.
	// Si falla (amount <= 0, categoría inválida, etc.), devolvemos el error al cliente.
	if err := h.serv.Create(&expense); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Todo salió bien — respondemos 201 Created (sin body, solo el status)
	w.WriteHeader(http.StatusCreated)
}

// HandleDelete — DELETE /api/expenses/{id}
func (h *ExpenseHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	// 1. Sacamos el "id" de la URL. Si la ruta es /api/expenses/5, PathValue da "5"
	idStr := r.PathValue("id")

	// 2. Convertimos el string a int64.
	// ParseInt(string, base, bits) — base 10 (decimal), 64 bits
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid expense id", http.StatusBadRequest)
		return
	}

	// 3. Pedimos al service que lo elimine
	if err := h.serv.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Respondemos 200 OK (el default si no llamas WriteHeader)
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes conecta cada handler a su ruta en el mux.
// El formato "METHOD /ruta" es de Go 1.22+ — antes tenías que checar r.Method manualmente.
func (h *ExpenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/expenses", h.HandleList)
	mux.HandleFunc("POST /api/expenses", h.HandleCreate)
	mux.HandleFunc("DELETE /api/expenses/{id}", h.HandleDelete)
}
