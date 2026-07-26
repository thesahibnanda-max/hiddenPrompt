package handler

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/service/admin"
	"net/http"

	redisRate "github.com/go-redis/redis_rate/v10"
	"github.com/valyala/fasthttp"
)

// adminRateLimit is deliberately conservative and per-IP (pre-auth style,
// like signup/login) since there's no user identity involved - it exists
// to slow down brute-forcing the admin key, not to accommodate a
// legitimate high-volume caller.
var adminRateLimit = redisRate.PerHour(20)

type AdminHandler interface {
	ClearCache(ctx *fasthttp.RequestCtx)
	DeleteUser(ctx *fasthttp.RequestCtx)
}

type adminHandler struct {
	base
	service admin.Service
	db      *database.DatabaseParams
}

func NewAdminHandler(service admin.Service, db *database.DatabaseParams) (AdminHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &adminHandler{service: service, db: db}, nil
}

func (a *adminHandler) ClearCache(ctx *fasthttp.RequestCtx) {
	a.timeTakenStart(ctx)
	defer a.deferTimeTaken(ctx)

	if !a.enforceRateLimit(ctx, a.db.RedisRateLimiter, constants.RateLimitAdminKeyPrefix+a.clientIP(ctx), adminRateLimit) {
		return
	}

	if !a.requireAdminAuth(ctx, a.service) {
		return
	}

	if err := a.service.ClearDataFromCache(ctx); err != nil {
		a.writeServiceError(ctx, err)
		return
	}

	a.writeJSONResponse(ctx, http.StatusOK, successResponse{Message: "Cache cleared"})
}

func (a *adminHandler) DeleteUser(ctx *fasthttp.RequestCtx) {
	a.timeTakenStart(ctx)
	defer a.deferTimeTaken(ctx)

	if !a.enforceRateLimit(ctx, a.db.RedisRateLimiter, constants.RateLimitAdminKeyPrefix+a.clientIP(ctx), adminRateLimit) {
		return
	}

	if !a.requireAdminAuth(ctx, a.service) {
		return
	}

	req, err := bindBodyJSON[models.AdminDeleteUserRequest](ctx)
	if err != nil {
		a.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	if err := a.service.DeleteUserRelatedDataFromRelationalDBs(ctx, req.Email); err != nil {
		a.writeServiceError(ctx, err)
		return
	}

	a.writeJSONResponse(ctx, http.StatusOK, successResponse{Message: "User data deleted"})
}
