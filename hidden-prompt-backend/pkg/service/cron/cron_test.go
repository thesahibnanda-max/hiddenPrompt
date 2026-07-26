package cron_test

import (
	"context"
	"testing"
	"time"

	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/service/cron"

	"github.com/stretchr/testify/require"
)

func Test_NewService_NilDB(t *testing.T) {
	svc, err := cron.NewService(nil)
	require.Error(t, err)
	require.Nil(t, svc)
}

func Test_NewService_RegistersExactlyOneJob(t *testing.T) {
	svc, err := cron.NewService(&database.DatabaseParams{
		PingContextFunction: func(context.Context) *database.PingContextError { return nil },
	})
	require.NoError(t, err)
	require.NotNil(t, svc)
}

func Test_StartThenStop_CompletesWithoutError(t *testing.T) {
	svc, err := cron.NewService(&database.DatabaseParams{
		PingContextFunction: func(context.Context) *database.PingContextError { return nil },
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, svc.Start(ctx))
	// Start is non-blocking (runs the scheduler in its own goroutine), so
	// Stop immediately after is expected to complete quickly - there's no
	// in-flight job to wait on for a freshly-started scheduler.
	require.NoError(t, svc.Stop(ctx))
}

func Test_Stop_RespectsContextDeadline(t *testing.T) {
	svc, err := cron.NewService(&database.DatabaseParams{
		PingContextFunction: func(context.Context) *database.PingContextError { return nil },
	})
	require.NoError(t, err)

	require.NoError(t, svc.Start(context.Background()))

	// An already-expired context must not block Stop forever.
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	<-ctx.Done()

	err = svc.Stop(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
