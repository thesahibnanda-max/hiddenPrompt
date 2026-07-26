package deps

import (
	"context"
	"hidden-prompt-backend/pkg/algo"
	chatAndAIUsage "hidden-prompt-backend/pkg/chat"
	geminiClient "hidden-prompt-backend/pkg/clients/gemini"
	groqClient "hidden-prompt-backend/pkg/clients/groq"
	"hidden-prompt-backend/pkg/core/middleware"
	"hidden-prompt-backend/pkg/core/router"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/handler"
	"hidden-prompt-backend/pkg/mail"
	"hidden-prompt-backend/pkg/service/admin"
	"hidden-prompt-backend/pkg/service/cron"
	"hidden-prompt-backend/pkg/service/intent"
	"hidden-prompt-backend/pkg/service/music"
	"hidden-prompt-backend/pkg/service/puzzle"
	"hidden-prompt-backend/pkg/service/user"
	"log/slog"
	"net"
	"strconv"

	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

const defaultPort = 8080

// NewApp assembles the full dependency graph - config, clients, database,
// services, handlers, middleware, and the router - and registers the
// fasthttp server's lifecycle. Nothing runs until the returned app is
// started (via Run, or manually via app.Start/app.Stop for tests).
func NewApp(populateOptions ...fx.Option) *fx.App {
	opts := []fx.Option{
		fx.Provide(
			database.NewDatabase,
			groqClient.NewClient,
			geminiClient.NewClient,
			mail.NewMailer,
			algo.NewAlgorithm,
			chatAndAIUsage.NewAI,
			user.NewService,
			intent.NewService,
			puzzle.NewService,
			admin.NewService,
			music.NewService,
			cron.NewService,
			handler.NewUserHandler,
			handler.NewPuzzleHandler,
			handler.NewHealthHandler,
			handler.NewAdminHandler,
			handler.NewMusicHandler,
			middleware.NewMiddleware,
			router.NewRouter,
		),

		fx.Invoke(
			registerHTTPServer,
			registerCronService,
		),

		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.Default()}
		}),
	}

	if len(populateOptions) > 0 {
		for _, opt := range populateOptions {
			if opt == nil {
				continue
			}
			opts = append(opts, opt)
		}
	}

	return fx.New(opts...)
}

// registerHTTPServer wires the assembled Router into a *fasthttp.Server and
// hooks its start/stop into the fx lifecycle, so the server comes up after
// every dependency is ready and shuts down gracefully (draining in-flight
// connections) when the app stops.
func registerHTTPServer(lc fx.Lifecycle, r router.Router) {
	server := &fasthttp.Server{
		Handler: r.Handler(),
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			addr := ":" + strconv.Itoa(defaultPort)

			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			go func() {
				if err := server.Serve(ln); err != nil {
					slog.Error("fasthttp server stopped unexpectedly", slog.Any("error", err))
				}
			}()

			slog.Info("server started", slog.String("addr", addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.ShutdownWithContext(ctx)
		},
	})
}

// registerCronService hooks the cron service's Start/Stop into the fx
// lifecycle alongside the HTTP server, so scheduled jobs run once the app
// starts and shut down gracefully (waiting for any in-flight job) when it
// stops. Both methods already match fx.Hook's field signatures directly.
func registerCronService(lc fx.Lifecycle, c cron.Service) {
	lc.Append(fx.Hook{
		OnStart: c.Start,
		OnStop:  c.Stop,
	})
}
