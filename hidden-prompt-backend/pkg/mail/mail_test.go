package mail

import (
	"context"
	"hidden-prompt-backend/pkg/utils"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewMailer(t *testing.T) {
	m, err := NewMailer()
	require.NoError(t, err)
	require.NotNil(t, m)

	otp, err := utils.GenerateRandomString(4)
	require.NoError(t, err)
	require.NoError(t, m.SendOTPMail(context.Background(), "sabbykabby12@gmail.com", otp))
}
