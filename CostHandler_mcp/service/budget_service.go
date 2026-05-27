package service

import (
	"fmt"
	"math"
	"slices"

	"github.com/GuillermoSego/costhandler/mcp/models"
	"github.com/GuillermoSego/costhandler/mcp/repository"
)

type BudgetService struct {
	repo repository.BudgetRepository
}

func NewBudgetService(repo repository.BudgetRepository) *BudgetService {
	return &BudgetService{repo: repo}
}

func (s *BudgetService) Upsert(budget *models.Budget) error {
	if budget.Amount <= 0 {
		return fmt.Errorf("invalid budget amount: must be greater than zero")
	}
	if !slices.Contains(validCategories, budget.Category) {
		return fmt.Errorf("invalid category: %s", budget.Category)
	}
	if budget.User == "" {
		return fmt.Errorf("user is required")
	}
	return s.repo.Upsert(budget)
}

func (s *BudgetService) ListByUser(user string) ([]models.Budget, error) {
	return s.repo.ListByUser(user)
}

func (s *BudgetService) Delete(user, category string) error {
	return s.repo.Delete(user, category)
}

func (s *BudgetService) CompareBudget(user string, byCategory []models.CategorySummary) ([]models.BudgetComparison, error) {
	budgets, err := s.repo.ListByUser(user)
	if err != nil {
		return nil, err
	}
	if len(budgets) == 0 {
		return nil, nil
	}

	spentMap := make(map[string]float64)
	for _, c := range byCategory {
		spentMap[c.Category] = c.Total
	}

	var comparisons []models.BudgetComparison
	for _, b := range budgets {
		spent := spentMap[b.Category]
		pct := 0.0
		if b.Amount > 0 {
			pct = math.Round((spent/b.Amount)*10000) / 100
		}
		comparisons = append(comparisons, models.BudgetComparison{
			Category:   b.Category,
			Budgeted:   b.Amount,
			Spent:      math.Round(spent*100) / 100,
			Remaining:  math.Round((b.Amount-spent)*100) / 100,
			Percentage: pct,
		})
	}
	return comparisons, nil
}

func (s *BudgetService) UpsertChatID(user string, chatID int64) error {
	return s.repo.UpsertChatID(user, chatID)
}

func (s *BudgetService) ListChatIDs() ([]models.UserChat, error) {
	return s.repo.ListChatIDs()
}
