package models_test

import (
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestNewCompletionChunk(t *testing.T) {
	finishReason := "stop"
	chunk := models.NewCompletionChunk("chatcmpl-123", "gpt-4o-mini", "Hello", &finishReason)

	assert.Equal(t, "chatcmpl-123", chunk.ID)
	assert.Equal(t, "chat.completion.chunk", chunk.Object)
	assert.Equal(t, "gpt-4o-mini", chunk.Model)
	assert.Len(t, chunk.Choices, 1)
	assert.Equal(t, "Hello", chunk.Choices[0].Delta.Content)
	assert.Equal(t, &finishReason, chunk.Choices[0].FinishReason)
}

func TestCompletionChunk_IsComplete(t *testing.T) {
	tests := []struct {
		name  string
		chunk *models.CompletionChunk
		want  bool
	}{
		{
			name: "complete with finish_reason",
			chunk: &models.CompletionChunk{
				Choices: []models.ChunkChoice{
					{FinishReason: strPtr("stop")},
				},
			},
			want: true,
		},
		{
			name: "incomplete without finish_reason",
			chunk: &models.CompletionChunk{
				Choices: []models.ChunkChoice{
					{FinishReason: nil},
				},
			},
			want: false,
		},
		{
			name:  "no choices",
			chunk: &models.CompletionChunk{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.chunk.IsComplete()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompletionChunk_GetContent(t *testing.T) {
	tests := []struct {
		name  string
		chunk *models.CompletionChunk
		want  string
	}{
		{
			name: "with content",
			chunk: &models.CompletionChunk{
				Choices: []models.ChunkChoice{
					{Delta: models.ChunkDelta{Content: "Hello"}},
				},
			},
			want: "Hello",
		},
		{
			name:  "no choices",
			chunk: &models.CompletionChunk{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.chunk.GetContent()
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
