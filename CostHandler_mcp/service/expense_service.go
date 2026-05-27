package service

import (
	"crypto/rand"
	"fmt"
	"math"
	"slices"
	"time"

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

	if expense.CreatedAt == "" {
		expense.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}

	return s.repo.Create(expense)
}

func (s *ExpenseService) List(user string) ([]models.Expense, error) {
	return s.repo.List(user)
}

func (s *ExpenseService) Update(expense *models.Expense) error {
	if expense.ID <= 0 {
		return fmt.Errorf("invalid expense id: %d", expense.ID)
	}
	if expense.Description == "" {
		return fmt.Errorf("invalid expense: description is required")
	}
	if !slices.Contains(validCategories, expense.Category.Name) {
		return fmt.Errorf("invalid category: %s", expense.Category.Name)
	}
	if expense.CreatedAt == "" {
		return fmt.Errorf("invalid expense: date is required")
	}
	return s.repo.Update(expense)
}

// Delete elimina un gasto por ID.
func (s *ExpenseService) Delete(id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid expense id: %d", id)
	}
	return s.repo.Delete(id)
}

// ValidCategories expone la lista de categorías válidas para el dashboard.
func ValidCategories() []string {
	return validCategories
}

func (s *ExpenseService) ListFiltered(filter models.ExpenseFilter) ([]models.Expense, error) {
	if filter.Category != "" && !slices.Contains(validCategories, filter.Category) {
		return nil, fmt.Errorf("invalid category filter: %s", filter.Category)
	}
	return s.repo.ListFiltered(filter)
}

func (s *ExpenseService) CreateInstallments(expense *models.Expense, totalAmount float64, installments int) ([]*models.Expense, error) {
	if installments < 2 || installments > 48 {
		return nil, fmt.Errorf("número de mensualidades inválido: debe ser entre 2 y 48")
	}
	if totalAmount <= 0 {
		return nil, fmt.Errorf("monto inválido: debe ser mayor a cero")
	}
	if expense.Description == "" {
		return nil, fmt.Errorf("descripción requerida")
	}
	if !slices.Contains(validCategories, expense.Category.Name) {
		return nil, fmt.Errorf("categoría inválida: %s", expense.Category.Name)
	}

	perInstallment := math.Floor(totalAmount*100/float64(installments)) / 100
	remainder := math.Round((totalAmount-perInstallment*float64(installments))*100) / 100

	groupID := generateGroupID()
	now := time.Now()

	var expenses []*models.Expense
	for i := 0; i < installments; i++ {
		amt := perInstallment
		if i == 0 {
			amt += remainder
		}

		date := now.AddDate(0, i, 0)

		expenses = append(expenses, &models.Expense{
			User:              expense.User,
			Amount:            amt,
			Description:       fmt.Sprintf("%s (%d/%d)", expense.Description, i+1, installments),
			Category:          expense.Category,
			RawMessage:        expense.RawMessage,
			CreatedAt:         date.Format("2006-01-02 15:04:05"),
			InstallmentGroup:  groupID,
			InstallmentNumber: i + 1,
			TotalInstallments: installments,
		})
	}

	if err := s.repo.CreateBatch(expenses); err != nil {
		return nil, fmt.Errorf("guardando mensualidades: %w", err)
	}

	return expenses, nil
}

func (s *ExpenseService) ListDistinctUsers() ([]string, error) {
	return s.repo.ListDistinctUsers()
}

func generateGroupID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *ExpenseService) GetDashboardData(user, period, category string) (*models.DashboardData, error) {
	now := time.Now()
	from, to := periodToRange(period, now)

	byCategory, err := s.repo.SumByCategory(user, from, to)
	if err != nil {
		return nil, err
	}

	byDay, err := s.repo.SumByDay(user, from, to)
	if err != nil {
		return nil, err
	}

	byMonth, err := s.repo.SumByMonth(user, 12)
	if err != nil {
		return nil, err
	}

	var totalAmount float64
	var expenseCount int
	var topCategory string
	var topCategoryAmt float64
	for _, c := range byCategory {
		totalAmount += c.Total
		expenseCount += c.Count
		if c.Total > topCategoryAmt {
			topCategoryAmt = c.Total
			topCategory = c.Category
		}
	}

	days := daysBetween(from, to)
	if days < 1 {
		days = 1
	}
	dailyAverage := totalAmount / float64(days)

	prevFrom, prevTo := prevPeriodRange(period, now)
	prevByCategory, err := s.repo.SumByCategory(user, prevFrom, prevTo)
	if err != nil {
		return nil, err
	}
	var prevTotal float64
	for _, c := range prevByCategory {
		prevTotal += c.Total
	}

	return &models.DashboardData{
		TotalAmount:    math.Round(totalAmount*100) / 100,
		DailyAverage:   math.Round(dailyAverage*100) / 100,
		TopCategory:    topCategory,
		TopCategoryAmt: math.Round(topCategoryAmt*100) / 100,
		PrevTotal:      math.Round(prevTotal*100) / 100,
		ByCategory:     byCategory,
		ByDay:          byDay,
		ByMonth:        byMonth,
		ExpenseCount:   expenseCount,
	}, nil
}

func periodToRange(period string, now time.Time) (string, string) {
	to := now.Format("2006-01-02")
	var from time.Time
	switch period {
	case "week":
		from = now.AddDate(0, 0, -7)
	case "year":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	return from.Format("2006-01-02"), to
}

func prevPeriodRange(period string, now time.Time) (string, string) {
	switch period {
	case "week":
		to := now.AddDate(0, 0, -7).Format("2006-01-02")
		from := now.AddDate(0, 0, -14).Format("2006-01-02")
		return from, to
	case "year":
		prevYear := now.Year() - 1
		from := time.Date(prevYear, 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		to := time.Date(prevYear, 12, 31, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		return from, to
	default:
		prevMonth := now.AddDate(0, -1, 0)
		from := time.Date(prevMonth.Year(), prevMonth.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		lastDay := time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, now.Location())
		return from, lastDay.Format("2006-01-02")
	}
}

func daysBetween(from, to string) int {
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil {
		return 1
	}
	days := int(t.Sub(f).Hours()/24) + 1
	if days < 1 {
		return 1
	}
	return days
}
