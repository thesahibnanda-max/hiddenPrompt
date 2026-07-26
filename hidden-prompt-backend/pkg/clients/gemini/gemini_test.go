package geminiClient_test

import (
	"context"
	geminiClient "hidden-prompt-backend/pkg/clients/gemini"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetEmbedding(t *testing.T) {
	c, err := geminiClient.NewClient()
	require.NoError(t, err)
	resp, err := c.GetEmbeddings(context.Background(), "Write 42 in alphabets")
	
	require.NoError(t, err)
	t.Logf("%+v", len(resp))
}
