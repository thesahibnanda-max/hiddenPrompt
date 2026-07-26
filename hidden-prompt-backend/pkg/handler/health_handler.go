package handler

import (
	"fmt"
	"hidden-prompt-backend/pkg/database"
	"log/slog"
	"net/http"

	"github.com/valyala/fasthttp"
)

type HealthHandler interface {
	GetHealth(ctx *fasthttp.RequestCtx)
}

type healthHandler struct {
	base
	db *database.DatabaseParams
}

func NewHealthHandler(db *database.DatabaseParams) (HealthHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &healthHandler{db: db}, nil
}

func (h *healthHandler) GetHealth(ctx *fasthttp.RequestCtx) {
	h.timeTakenStart(ctx)
	defer h.deferTimeTaken(ctx)

	// Health checks are polled frequently by uptime monitors, load
	// balancers, and orchestrators - rate limiting this endpoint would make
	// those probes themselves look like an outage, so it's intentionally
	// excluded from enforceRateLimit.

	pingErr := h.db.PingContextFunction(ctx)
	if pingErr != nil {
		errs := make(map[string]string)
		if pingErr.MySQLPingErr != nil {
			errs["mysql"] = pingErr.MySQLPingErr.Error()
			slog.Error("health check: mysql ping failed", slog.Any("error", pingErr.MySQLPingErr))
		}
		if pingErr.PostgreSQLErr != nil {
			errs["postgresql"] = pingErr.PostgreSQLErr.Error()
			slog.Error("health check: postgresql ping failed", slog.Any("error", pingErr.PostgreSQLErr))
		}
		if pingErr.RedisErr != nil {
			errs["redis"] = pingErr.RedisErr.Error()
			slog.Error("health check: redis ping failed", slog.Any("error", pingErr.RedisErr))
		}
		if pingErr.MongoErr != nil {
			errs["mongodb"] = pingErr.MongoErr.Error()
			slog.Error("health check: mongodb ping failed", slog.Any("error", pingErr.MongoErr))
		}

		// Not every non-nil error field necessarily means every value was
		// populated above - guard against writing an empty map if
		// PingContextFunction ever returns a non-nil struct with all-nil
		// fields, which would otherwise report "unhealthy" with no reason.
		if len(errs) == 0 {
			errs["unknown"] = "ping reported unhealthy but no specific dependency error was set"
			slog.Error("health check: ping returned a non-nil error with no populated fields")
		}

		h.writeJSONResponse(ctx, http.StatusServiceUnavailable, healthCheckResponse{
			Status: "unhealthy",
			Errors: errs,
		})
		return
	}

	h.writeJSONResponse(ctx, http.StatusOK, healthCheckResponse{Status: "ok"})
}
