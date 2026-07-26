package groqClient

import (
	"context"
	"testing"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/stretchr/testify/require"
)

func Test_ClientCall(t *testing.T) {
	c, err := NewClient()
	require.NoError(t, err)

	resp, err := c.Call(context.Background(), CallRequest{
		SystemPrompt: "REPLY AS A GF",
		UserPrompt:   "Say hello in one short sentence.",
		// ModelToUse:   lo.ToPtr(constants.GPTOSS20B),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Response)

	t.Logf("%s", goxJsonUtils.PrettyStringSuppressError(resp))
}
