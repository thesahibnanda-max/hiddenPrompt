package config_test

import (
	"hidden-prompt-backend/pkg/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetConfig(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)

	t.Logf("%+v", cfg.MailConnections)
}
