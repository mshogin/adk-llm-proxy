package models_test

import (
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestCompletionRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CompletionRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &models.CompletionRequest{
				Model: "gpt-4o-mini",
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing model",
			req: &models.CompletionRequest{
				Messages: []models.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: models.ErrMissingModel,
		},
		{
			name: "empty messages",
			req: &models.CompletionRequest{
				Model:    "gpt-4o-mini",
				Messages: []models.Message{},
			},
			wantErr: models.ErrEmptyMessages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompletionRequest_GetTemperature(t *testing.T) {
	tests := []struct {
		name string
		req  *models.CompletionRequest
		want float64
	}{
		{
			name: "with temperature",
			req: &models.CompletionRequest{
				Temperature: floatPtr(0.7),
			},
			want: 0.7,
		},
		{
			name: "without temperature (default)",
			req:  &models.CompletionRequest{},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.GetTemperature()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletionRequest_GetMaxTokens(t *testing.T) {
	tests := []struct {
		name string
		req  *models.CompletionRequest
		want int
	}{
		{
			name: "with max tokens",
			req: &models.CompletionRequest{
				MaxTokens: intPtr(100),
			},
			want: 100,
		},
		{
			name: "without max tokens (default)",
			req:  &models.CompletionRequest{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.GetMaxTokens()
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
