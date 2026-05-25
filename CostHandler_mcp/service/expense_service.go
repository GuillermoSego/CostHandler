package service

import (
	"fmt"
	"slices"

	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/repository"
)

type ExpenseService struct {
	repo repository.ExpenseRepository
}

// validCategories vive a nivel de paquete porque no cambia entre llamadas.
var validCategories = []string{
	"supermercado",    // Costco, Walmart, Bodega Aurrera, Soriana, carnicería, frutas
	"restaurantes",    // Uber Eats, Rappi, tacos, Starbucks, comida fuera
	"vivienda",        // Renta, reparaciones hogar, carpintero, ferretería
	"servicios",       // Luz, gas, agua, internet, celular, filtro
	"transporte",      // Gasolina, casetas, Uber/Didi, mantenimiento auto, llantas
	"salud",           // Farmacia, doctor, pediatra, terapia, lentes
	"familia",         // Nani, guardería, fórmula, pañales, educación hijos
	"suscripciones",   // Spotify, ChatGPT, Claude, apps
	"entretenimiento", // Vacaciones, cine, hoteles, salidas recreativas
	"compras",         // Ropa, IKEA, Amazon, Temu, muebles, tecnología
	"ahorro",          // Fondo emergencia, retiro, fibras
	"otros",           // Regalos, impuestos, deuda, misceláneos
}

func NewExpenseService(repo repository.ExpenseRepository) *ExpenseService {
	return &ExpenseService{repo: repo}
}

// Create valida el gasto y lo guarda via el repository.
func (s *ExpenseService) Create(expense *models.Expense) error {
	if expense.Amount <= 0 {
		return fmt.Errorf("invalid amount: must be greater than zero")
	}
	if expense.Description == "" {
		return fmt.Errorf("invalid expense: description is required")
	}
	if !slices.Contains(validCategories, expense.Category.Name) {
		return fmt.Errorf("invalid category: %s", expense.Category.Name)
	}

	// Validaciones pasaron — delegamos al repo
	return s.repo.Create(expense)
}

// List devuelve todos los gastos.
func (s *ExpenseService) List() ([]models.Expense, error) {
	return s.repo.List()
}

// Delete elimina un gasto por ID.
func (s *ExpenseService) Delete(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid expense id: %d", id)
	}
	return s.repo.Delete(id)
}
