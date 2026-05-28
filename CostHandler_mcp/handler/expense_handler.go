package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/service"
)

type ExpenseHandler struct {
	serv      *service.ExpenseService
	budgetSvc *service.BudgetService
}

func NewExpenseHandler(serv *service.ExpenseService, budgetSvc *service.BudgetService) *ExpenseHandler {
	return &ExpenseHandler{serv: serv, budgetSvc: budgetSvc}
}

// HandleList — GET /api/expenses
// Soporta query params opcionales: ?period=month&category=restaurantes
func (h *ExpenseHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	category := r.URL.Query().Get("category")
	user := r.URL.Query().Get("user")

	var expenses []models.Expense
	var err error

	if period != "" || category != "" {
		filter := models.ExpenseFilter{
			User:     user,
			Period:   period,
			Category: category,
			From:     r.URL.Query().Get("from"),
			To:       r.URL.Query().Get("to"),
		}
		if filter.Period == "" {
			filter.Period = "month"
		}
		expenses, err = h.serv.ListFiltered(filter)
	} else {
		expenses, err = h.serv.List(user)
	}

	if err != nil {
		http.Error(w, "error listing expenses", http.StatusInternalServerError)
		return
	}

	if expenses == nil {
		expenses = []models.Expense{}
	}

	w.Header().Set("Content-Type", "application/json")
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

// HandleUpdate — PUT /api/expenses/{id}
func (h *ExpenseHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid expense id", http.StatusBadRequest)
		return
	}

	var expense models.Expense
	if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	expense.ID = id

	if err := h.serv.Update(&expense); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleDashboardSummary — GET /api/dashboard/summary
func (h *ExpenseHandler) HandleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	user := r.URL.Query().Get("user")
	category := r.URL.Query().Get("category")
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")

	if period == "" {
		period = "month"
	}

	data, err := h.serv.GetDashboardData(user, period, category, fromParam, toParam)
	if err != nil {
		http.Error(w, "error generating dashboard summary", http.StatusInternalServerError)
		return
	}

	if user != "" && h.budgetSvc != nil {
		comparison, err := h.budgetSvc.CompareBudget(user, data.ByCategory)
		if err == nil && comparison != nil {
			data.BudgetComparison = comparison
			for _, c := range comparison {
				data.TotalBudgeted += c.Budgeted
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *ExpenseHandler) HandleBudgetList(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "user parameter required", http.StatusBadRequest)
		return
	}

	budgets, err := h.budgetSvc.ListByUser(user)
	if err != nil {
		http.Error(w, "error listing budgets", http.StatusInternalServerError)
		return
	}
	if budgets == nil {
		budgets = []models.Budget{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(budgets)
}

func (h *ExpenseHandler) HandleBudgetUpsert(w http.ResponseWriter, r *http.Request) {
	var budget models.Budget
	if err := json.NewDecoder(r.Body).Decode(&budget); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.budgetSvc.Upsert(&budget); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *ExpenseHandler) HandleUserList(w http.ResponseWriter, r *http.Request) {
	users, err := h.serv.ListDistinctUsers()
	if err != nil {
		http.Error(w, "error listing users", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *ExpenseHandler) HandleInstallments(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	data, err := h.serv.GetInstallmentSummary(user)
	if err != nil {
		http.Error(w, "error fetching installment data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *ExpenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/expenses", h.HandleList)
	mux.HandleFunc("POST /api/expenses", h.HandleCreate)
	mux.HandleFunc("PUT /api/expenses/{id}", h.HandleUpdate)
	mux.HandleFunc("DELETE /api/expenses/{id}", h.HandleDelete)
	mux.HandleFunc("GET /api/dashboard/summary", h.HandleDashboardSummary)
	mux.HandleFunc("GET /api/budgets", h.HandleBudgetList)
	mux.HandleFunc("POST /api/budgets", h.HandleBudgetUpsert)
	mux.HandleFunc("GET /api/users", h.HandleUserList)
	mux.HandleFunc("GET /api/installments", h.HandleInstallments)
}
