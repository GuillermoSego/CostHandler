package service

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/GuillermoSego/costhandler/mcp/database"
	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/repository"
)

// setupTestService crea un service con SQLite in-memory para cada test.
// ":memory:" = base de datos que vive solo en RAM, se destruye al terminar.
// Así cada test empieza con una DB limpia, sin residuos de otros tests.
func setupTestService(t *testing.T) *ExpenseService {
	// t.Helper() le dice a Go: "si algo falla aquí, reporta el error
	// en la línea del test que llamó a setup, no aquí dentro".
	t.Helper()

	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	if err := database.CreateTables(db); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	repo := repository.NewSQLiteExpenseRepository(db)
	return NewExpenseService(repo)
}

// validExpense devuelve un Expense válido como base.
// Los tests que prueban UNA validación cambian solo ese campo.
func validExpense() models.Expense {
	return models.Expense{
		User:        "guillermo",
		Amount:      150.0,
		Description: "Tacos al pastor",
		Category:    models.Category{Name: "restaurantes"},
		RawMessage:  "150 tacos",
	}
}

// TestCreate usa "table-driven tests" — el patrón más común en Go.
// Defines una tabla con casos, los iteras, y cada uno corre como sub-test.
func TestCreate(t *testing.T) {
	// Cada elemento de la tabla es un caso de prueba.
	// "name" aparece en la salida si el test falla, para que sepas cuál fue.
	tests := []struct {
		name    string                // Nombre descriptivo del caso
		modify  func(*models.Expense) // Función que modifica el expense válido
		wantErr bool                  // ¿Esperamos que falle?
	}{
		{
			name:    "valid expense should succeed",
			modify:  func(e *models.Expense) {}, // No modificamos nada — debe pasar
			wantErr: false,
		},
		{
			name:    "zero amount should fail",
			modify:  func(e *models.Expense) { e.Amount = 0 },
			wantErr: true,
		},
		{
			name:    "negative amount should fail",
			modify:  func(e *models.Expense) { e.Amount = -50 },
			wantErr: true,
		},
		{
			name:    "empty description should fail",
			modify:  func(e *models.Expense) { e.Description = "" },
			wantErr: true,
		},
		{
			name:    "invalid category should fail",
			modify:  func(e *models.Expense) { e.Category.Name = "pizza" },
			wantErr: true,
		},
		{
			name:    "valid category supermercado should succeed",
			modify:  func(e *models.Expense) { e.Category.Name = "supermercado" },
			wantErr: false,
		},
	}

	// range itera sobre la tabla. Cada "tt" es un caso de prueba.
	for _, tt := range tests {
		// t.Run crea un sub-test con nombre. En la salida verás:
		// TestCreate/valid_expense_should_succeed  ✓
		// TestCreate/zero_amount_should_fail       ✓
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTestService(t)
			expense := validExpense()
			tt.modify(&expense) // Aplicamos la modificación de este caso

			err := svc.Create(&expense)

			// Esta es la aserción: comparamos si GOT error vs WANT error
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestList verifica que los gastos guardados se puedan recuperar.
func TestList(t *testing.T) {
	svc := setupTestService(t)

	expenses, err := svc.List("guillermo")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 0 {
		t.Errorf("expected 0 expenses, got %d", len(expenses))
	}

	e1 := validExpense()
	e2 := validExpense()
	e2.Description = "Uber al trabajo"
	e2.Category = models.Category{Name: "transporte"}
	e2.Amount = 85.0

	svc.Create(&e1)
	svc.Create(&e2)

	expenses, err = svc.List("guillermo")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 2 {
		t.Errorf("expected 2 expenses, got %d", len(expenses))
	}

	expenses, err = svc.List("otro_usuario")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 0 {
		t.Errorf("expected 0 expenses for other user, got %d", len(expenses))
	}
}

// TestDelete verifica que se pueda eliminar un gasto.
func TestDelete(t *testing.T) {
	svc := setupTestService(t)

	// Creamos un gasto
	expense := validExpense()
	svc.Create(&expense)

	expenses, _ := svc.List("guillermo")
	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(expenses))
	}

	err := svc.Delete(expenses[0].ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	expenses, _ = svc.List("guillermo")
	if len(expenses) != 0 {
		t.Errorf("expected 0 expenses after delete, got %d", len(expenses))
	}
}

// TestDelete_InvalidID verifica que IDs inválidos son rechazados.
func TestDelete_InvalidID(t *testing.T) {
	svc := setupTestService(t)

	err := svc.Delete(0)
	if err == nil {
		t.Error("Delete(0) should return error")
	}

	err = svc.Delete(-5)
	if err == nil {
		t.Error("Delete(-5) should return error")
	}
}

func TestListFiltered(t *testing.T) {
	svc := setupTestService(t)

	e1 := validExpense()
	e2 := validExpense()
	e2.Description = "Uber"
	e2.Category = models.Category{Name: "transporte"}
	e2.Amount = 85.0

	svc.Create(&e1)
	svc.Create(&e2)

	t.Run("filter by category", func(t *testing.T) {
		filter := models.ExpenseFilter{User: "guillermo", Period: "month", Category: "restaurantes"}
		expenses, err := svc.ListFiltered(filter)
		if err != nil {
			t.Fatalf("ListFiltered() error = %v", err)
		}
		if len(expenses) != 1 {
			t.Errorf("expected 1 expense, got %d", len(expenses))
		}
	})

	t.Run("filter all categories", func(t *testing.T) {
		filter := models.ExpenseFilter{User: "guillermo", Period: "month"}
		expenses, err := svc.ListFiltered(filter)
		if err != nil {
			t.Fatalf("ListFiltered() error = %v", err)
		}
		if len(expenses) != 2 {
			t.Errorf("expected 2 expenses, got %d", len(expenses))
		}
	})

	t.Run("filter by different user returns empty", func(t *testing.T) {
		filter := models.ExpenseFilter{User: "otro_usuario", Period: "month"}
		expenses, err := svc.ListFiltered(filter)
		if err != nil {
			t.Fatalf("ListFiltered() error = %v", err)
		}
		if len(expenses) != 0 {
			t.Errorf("expected 0 expenses for other user, got %d", len(expenses))
		}
	})

	t.Run("invalid category returns error", func(t *testing.T) {
		filter := models.ExpenseFilter{Period: "month", Category: "pizza"}
		_, err := svc.ListFiltered(filter)
		if err == nil {
			t.Error("expected error for invalid category filter")
		}
	})
}

func TestGetDashboardData(t *testing.T) {
	svc := setupTestService(t)

	e1 := validExpense()
	e1.Amount = 200
	e2 := validExpense()
	e2.Description = "Uber"
	e2.Category = models.Category{Name: "transporte"}
	e2.Amount = 100

	svc.Create(&e1)
	svc.Create(&e2)

	data, err := svc.GetDashboardData("guillermo", "month", "")
	if err != nil {
		t.Fatalf("GetDashboardData() error = %v", err)
	}

	if data.TotalAmount != 300 {
		t.Errorf("expected total 300, got %f", data.TotalAmount)
	}
	if data.TopCategory != "restaurantes" {
		t.Errorf("expected top category restaurantes, got %s", data.TopCategory)
	}
	if data.ExpenseCount != 2 {
		t.Errorf("expected 2 expenses, got %d", data.ExpenseCount)
	}
	if len(data.ByCategory) != 2 {
		t.Errorf("expected 2 categories, got %d", len(data.ByCategory))
	}
	if data.DailyAverage <= 0 {
		t.Errorf("expected positive daily average, got %f", data.DailyAverage)
	}
}

func TestCreateInstallments(t *testing.T) {
	tests := []struct {
		name         string
		totalAmount  float64
		installments int
		wantErr      bool
		wantCount    int
	}{
		{
			name:         "6 meses sin intereses",
			totalAmount:  5000,
			installments: 6,
			wantCount:    6,
		},
		{
			name:         "12 meses",
			totalAmount:  12000,
			installments: 12,
			wantCount:    12,
		},
		{
			name:         "3 meses con residuo",
			totalAmount:  100,
			installments: 3,
			wantCount:    3,
		},
		{
			name:         "monto indivisible",
			totalAmount:  10,
			installments: 3,
			wantCount:    3,
		},
		{
			name:         "1 mensualidad debe fallar",
			totalAmount:  1000,
			installments: 1,
			wantErr:      true,
		},
		{
			name:         "0 mensualidades debe fallar",
			totalAmount:  1000,
			installments: 0,
			wantErr:      true,
		},
		{
			name:         "49 mensualidades debe fallar",
			totalAmount:  1000,
			installments: 49,
			wantErr:      true,
		},
		{
			name:         "monto cero debe fallar",
			totalAmount:  0,
			installments: 6,
			wantErr:      true,
		},
		{
			name:         "monto negativo debe fallar",
			totalAmount:  -500,
			installments: 6,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTestService(t)
			expense := &models.Expense{
				User:        "guillermo",
				Description: "Audífonos",
				Category:    models.Category{Name: "compras"},
				RawMessage:  "5000 audífonos a 6 meses",
			}

			result, err := svc.CreateInstallments(expense, tt.totalAmount, tt.installments)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateInstallments() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("expected %d installments, got %d", tt.wantCount, len(result))
			}
		})
	}
}

func TestCreateInstallments_AmountSum(t *testing.T) {
	svc := setupTestService(t)
	expense := &models.Expense{
		User:        "guillermo",
		Description: "Audífonos",
		Category:    models.Category{Name: "compras"},
		RawMessage:  "5000 audífonos a 6 meses",
	}

	result, err := svc.CreateInstallments(expense, 5000, 6)
	if err != nil {
		t.Fatalf("CreateInstallments() error = %v", err)
	}

	var total float64
	for _, e := range result {
		total += e.Amount
	}
	total = math.Round(total*100) / 100

	if total != 5000 {
		t.Errorf("sum of installments = %.2f, want 5000.00", total)
	}
}

func TestCreateInstallments_DateProgression(t *testing.T) {
	svc := setupTestService(t)
	expense := &models.Expense{
		User:        "guillermo",
		Description: "Laptop",
		Category:    models.Category{Name: "compras"},
		RawMessage:  "20000 laptop a 12 meses",
	}

	result, err := svc.CreateInstallments(expense, 20000, 12)
	if err != nil {
		t.Fatalf("CreateInstallments() error = %v", err)
	}

	now := time.Now()
	for i, e := range result {
		expectedDate := now.AddDate(0, i, 0).Format("2006-01")
		gotDate := e.CreatedAt[:7]
		if gotDate != expectedDate {
			t.Errorf("installment %d: expected month %s, got %s", i+1, expectedDate, gotDate)
		}
	}
}

func TestCreateInstallments_GroupAndNumbers(t *testing.T) {
	svc := setupTestService(t)
	expense := &models.Expense{
		User:        "guillermo",
		Description: "TV",
		Category:    models.Category{Name: "compras"},
		RawMessage:  "15000 tv a 3 meses",
	}

	result, err := svc.CreateInstallments(expense, 15000, 3)
	if err != nil {
		t.Fatalf("CreateInstallments() error = %v", err)
	}

	groupID := result[0].InstallmentGroup
	if groupID == "" {
		t.Fatal("expected non-empty installment group ID")
	}

	for i, e := range result {
		if e.InstallmentGroup != groupID {
			t.Errorf("installment %d: group = %s, want %s", i+1, e.InstallmentGroup, groupID)
		}
		if e.InstallmentNumber != i+1 {
			t.Errorf("installment %d: number = %d, want %d", i+1, e.InstallmentNumber, i+1)
		}
		if e.TotalInstallments != 3 {
			t.Errorf("installment %d: total = %d, want 3", i+1, e.TotalInstallments)
		}
	}

	expenses, err := svc.List("guillermo")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 3 {
		t.Errorf("expected 3 expenses in DB, got %d", len(expenses))
	}
}

func TestCreate_SetsCreatedAt(t *testing.T) {
	svc := setupTestService(t)
	expense := validExpense()

	if err := svc.Create(&expense); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	expenses, _ := svc.List("guillermo")
	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(expenses))
	}

	if expenses[0].CreatedAt == "" {
		t.Error("expected CreatedAt to be set, got empty string")
	}

	today := time.Now().Format("2006-01-02")
	if !strings.HasPrefix(expenses[0].CreatedAt, today) {
		t.Errorf("expected CreatedAt to start with %s, got %s", today, expenses[0].CreatedAt)
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*models.Expense)
		wantErr bool
	}{
		{
			name:    "valid update should succeed",
			modify:  func(e *models.Expense) {},
			wantErr: false,
		},
		{
			name:    "invalid id should fail",
			modify:  func(e *models.Expense) { e.ID = 0 },
			wantErr: true,
		},
		{
			name:    "negative id should fail",
			modify:  func(e *models.Expense) { e.ID = -1 },
			wantErr: true,
		},
		{
			name:    "empty description should fail",
			modify:  func(e *models.Expense) { e.Description = "" },
			wantErr: true,
		},
		{
			name:    "invalid category should fail",
			modify:  func(e *models.Expense) { e.Category.Name = "pizza" },
			wantErr: true,
		},
		{
			name:    "empty created_at should fail",
			modify:  func(e *models.Expense) { e.CreatedAt = "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTestService(t)

			orig := validExpense()
			svc.Create(&orig)
			expenses, _ := svc.List("guillermo")
			id := expenses[0].ID

			update := models.Expense{
				ID:          id,
				Description: "Descripción actualizada",
				Category:    models.Category{Name: "transporte"},
				CreatedAt:   "2026-05-20 12:00:00",
			}
			tt.modify(&update)

			err := svc.Update(&update)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdate_VerifiesChanges(t *testing.T) {
	svc := setupTestService(t)

	orig := validExpense()
	svc.Create(&orig)
	expenses, _ := svc.List("guillermo")
	id := expenses[0].ID

	update := &models.Expense{
		ID:          id,
		Description: "Uber al aeropuerto",
		Category:    models.Category{Name: "transporte"},
		CreatedAt:   "2026-05-20 14:30:00",
	}
	if err := svc.Update(update); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	expenses, _ = svc.List("guillermo")
	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(expenses))
	}
	e := expenses[0]
	if e.Description != "Uber al aeropuerto" {
		t.Errorf("description = %s, want 'Uber al aeropuerto'", e.Description)
	}
	if e.Category.Name != "transporte" {
		t.Errorf("category = %s, want 'transporte'", e.Category.Name)
	}
	if !strings.HasPrefix(e.CreatedAt, "2026-05-20") {
		t.Errorf("created_at = %s, want prefix '2026-05-20'", e.CreatedAt)
	}
}
