package handler

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/service/admin"
	"hidden-prompt-backend/pkg/service/music"
	"net/http"

	redisRate "github.com/go-redis/redis_rate/v10"
	"github.com/valyala/fasthttp"
)

var (
	// insertMusicRateLimit is per-IP (pre-auth style, gated by the admin
	// key rather than a user identity) - same reasoning and same value as
	// adminRateLimit in admin_handler.go: it exists to slow down
	// brute-forcing the admin key, not to accommodate a legitimate
	// high-volume caller.
	insertMusicRateLimit = redisRate.PerHour(20)
	// getAllMusicRateLimit is generous since this is a plain, unauthenticated
	// read a UI may poll - matches getPuzzleByIDRateLimit's sizing.
	getAllMusicRateLimit = redisRate.PerMinute(30)
)

type MusicHandler interface {
	InsertMusicDetails(ctx *fasthttp.RequestCtx)
	GetAllMusicDetails(ctx *fasthttp.RequestCtx)
}

type musicHandler struct {
	base
	service      music.Service
	adminService admin.Service
	db           *database.DatabaseParams
}

func NewMusicHandler(service music.Service, adminService admin.Service, db *database.DatabaseParams) (MusicHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}
	if adminService == nil {
		return nil, fmt.Errorf("adminService is nil")
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &musicHandler{service: service, adminService: adminService, db: db}, nil
}

func (m *musicHandler) InsertMusicDetails(ctx *fasthttp.RequestCtx) {
	m.timeTakenStart(ctx)
	defer m.deferTimeTaken(ctx)

	if !m.enforceRateLimit(ctx, m.db.RedisRateLimiter, constants.RateLimitInsertMusicKeyPrefix+m.clientIP(ctx), insertMusicRateLimit) {
		return
	}

	if !m.requireAdminAuth(ctx, m.adminService) {
		return
	}

	req, err := bindBodyJSON[[]models.MusicDetail](ctx)
	if err != nil {
		m.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	musicDetails, err := m.service.InsertMusicDetails(ctx, req)
	if err != nil {
		m.writeServiceError(ctx, err)
		return
	}

	m.writeJSONResponse(ctx, http.StatusCreated, musicDetails)
}

func (m *musicHandler) GetAllMusicDetails(ctx *fasthttp.RequestCtx) {
	m.timeTakenStart(ctx)
	defer m.deferTimeTaken(ctx)

	if !m.enforceRateLimit(ctx, m.db.RedisRateLimiter, constants.RateLimitGetAllMusicKeyPrefix+m.clientIP(ctx), getAllMusicRateLimit) {
		return
	}

	musicDetails, err := m.service.GetAllMusicDetails(ctx)
	if err != nil {
		m.writeServiceError(ctx, err)
		return
	}

	m.writeJSONResponse(ctx, http.StatusOK, musicDetails)
}
