package service

import (
	"testing"

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

	// Al inicio la lista debe estar vacía
	expenses, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 0 {
		t.Errorf("expected 0 expenses, got %d", len(expenses))
	}

	// Creamos dos gastos
	e1 := validExpense()
	e2 := validExpense()
	e2.Description = "Uber al trabajo"
	e2.Category = models.Category{Name: "transporte"}
	e2.Amount = 85.0

	svc.Create(&e1)
	svc.Create(&e2)

	// Ahora deben ser 2
	expenses, err = svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(expenses) != 2 {
		t.Errorf("expected 2 expenses, got %d", len(expenses))
	}
}

// TestDelete verifica que se pueda eliminar un gasto.
func TestDelete(t *testing.T) {
	svc := setupTestService(t)

	// Creamos un gasto
	expense := validExpense()
	svc.Create(&expense)

	// Verificamos que existe
	expenses, _ := svc.List()
	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(expenses))
	}

	// Lo eliminamos usando el ID que SQLite le asignó
	err := svc.Delete(expenses[0].ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verificamos que ya no existe
	expenses, _ = svc.List()
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
		filter := models.ExpenseFilter{Period: "month", Category: "restaurantes"}
		expenses, err := svc.ListFiltered(filter)
		if err != nil {
			t.Fatalf("ListFiltered() error = %v", err)
		}
		if len(expenses) != 1 {
			t.Errorf("expected 1 expense, got %d", len(expenses))
		}
	})

	t.Run("filter all categories", func(t *testing.T) {
		filter := models.ExpenseFilter{Period: "month"}
		expenses, err := svc.ListFiltered(filter)
		if err != nil {
			t.Fatalf("ListFiltered() error = %v", err)
		}
		if len(expenses) != 2 {
			t.Errorf("expected 2 expenses, got %d", len(expenses))
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

	data, err := svc.GetDashboardData("month", "")
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
