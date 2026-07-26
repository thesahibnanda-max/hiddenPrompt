package cron

import (
	"context"
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type service struct {
	cronService *cron.Cron
	db          *database.DatabaseParams
}

// Start implements [Service]. cron.Cron.Start runs the scheduler in its own
// goroutine and returns immediately - there's no error path from the
// library itself.
func (s *service) Start(ctx context.Context) error {
	s.cronService.Start()
	return nil
}

// Stop implements [Service]. cron.Cron.Stop signals the scheduler to stop
// and returns a context that's done once any currently-running job
// finishes; select against the caller's ctx so shutdown can still be
// bounded by an overall deadline instead of waiting forever on a stuck job.
func (s *service) Stop(ctx context.Context) error {
	stopped := s.cronService.Stop()
	select {
	case <-stopped.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func NewService(db *database.DatabaseParams) (Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	c := cron.New(
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
			cron.SkipIfStillRunning(cron.DefaultLogger),
		),
	)

	s := &service{
		db:          db,
		cronService: c,
	}

	if _, err := s.cronService.AddFunc(fmt.Sprintf("@every %s", constants.PingCronTime.String()), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := db.PingContextFunction(ctx)
		if err != nil {
			slog.Error("error in db Pings", slog.Any("pingErr", err)) // just log
		}
	}); err != nil {
		return nil, err
	}

	return s, nil
}
