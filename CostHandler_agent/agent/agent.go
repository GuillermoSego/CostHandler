package agent

import (
	"context"
	"fmt"

	"github.com/GuillermoSego/costhandler/agent/models"
	"github.com/GuillermoSego/costhandler/agent/openai"
)

type Agent struct {
	classifier *openai.Client
}

func NewAgent(classifier *openai.Client) *Agent {
	return &Agent{classifier: classifier}
}

func (a *Agent) ProcessMessage(ctx context.Context, user string, message string) (*models.ClassificationResult, error) {
	// Llamamos a OpenAI para clasificar el mensaje.
	// "result" es la VARIABLE, "ClassificationResult" es el TIPO.
	result, err := a.classifier.Classify(ctx, message)
	if err != nil {
		// Dos valores de retorno: (nil, error) cuando falla
		return nil, fmt.Errorf("classifying message: %w", err)
	}

	// Validaciones sobre lo que OpenAI devolvió
	if result.Amount <= 0 {
		return nil, fmt.Errorf("invalid amount: must be greater than zero")
	}
	if result.Confidence < 0.5 {
		return nil, fmt.Errorf("low confidence (%.2f): could not classify message reliably", result.Confidence)
	}

	return result, nil
}

func (a *Agent) GetInsights(ctx context.Context, prompt string) (string, error) {
	insights, err := a.classifier.GenerateInsights(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generating insights: %w", err)
	}
	return insights, nil
}
