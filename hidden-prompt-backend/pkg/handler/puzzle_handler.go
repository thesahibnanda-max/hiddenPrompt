package handler

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/service/puzzle"
	"net/http"

	redisRate "github.com/go-redis/redis_rate/v10"
	"github.com/valyala/fasthttp"
)

// Per-endpoint rate limits, tuned for the same resource-constrained (0.1
// CPU / 256MB) deployment as user_handler.go. All four are keyed by the
// authenticated user's email rather than IP - every request here already
// has a verified identity by the time enforceRateLimit runs, since auth is
// checked first.
var (
	// CreatePuzzleForUser is the most expensive call - 2 AI calls (a
	// combined question+answer generation, then an embedding) per
	// invocation against free-tier Groq/Gemini quotas, so it's kept the
	// tightest. Was previously 3 calls (separate hidden-prompt and
	// canonical-answer generation) at PerHour(5); merging those into one
	// call cut real per-puzzle cost by roughly a third, which is why this
	// is 8 rather than 5 - proportionally looser without raising actual
	// quota pressure versus before.
	createPuzzleRateLimit = redisRate.PerHour(8)
	// Plain reads, generous since a UI may poll them.
	getAllPuzzlesRateLimit = redisRate.PerMinute(20)
	getPuzzleByIDRateLimit = redisRate.PerMinute(30)
	// The core gameplay loop (every guess) - 2-3 AI calls per call
	// (embeddings, intent scoring, maybe a hint), capped tighter than the
	// reads but generous enough for real play pacing (~1 guess per 6s).
	addUserPromptRateLimit = redisRate.PerMinute(10)
)

type PuzzleHandler interface {
	CreatePuzzleForUser(ctx *fasthttp.RequestCtx)
	GetAllPuzzlesForUser(ctx *fasthttp.RequestCtx)
	GetPuzzleByID(ctx *fasthttp.RequestCtx)
	AddUserPrompt(ctx *fasthttp.RequestCtx)
}

type puzzleHandler struct {
	base
	service puzzle.Service
	db      *database.DatabaseParams
}

func NewPuzzleHandler(service puzzle.Service, db *database.DatabaseParams) (PuzzleHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &puzzleHandler{service: service, db: db}, nil
}

func (p *puzzleHandler) CreatePuzzleForUser(ctx *fasthttp.RequestCtx) {
	p.timeTakenStart(ctx)
	defer p.deferTimeTaken(ctx)

	email, ok := p.requireAuth(ctx)
	if !ok {
		return
	}

	if !p.enforceRateLimit(ctx, p.db.RedisRateLimiter, constants.RateLimitCreatePuzzleKeyPrefix+email, createPuzzleRateLimit) {
		return
	}

	puzzles, err := p.service.CreatePuzzleForUser(ctx, email)
	if err != nil {
		p.writeServiceError(ctx, err)
		return
	}

	p.writeJSONResponse(ctx, http.StatusCreated, puzzles)
}

func (p *puzzleHandler) GetAllPuzzlesForUser(ctx *fasthttp.RequestCtx) {
	p.timeTakenStart(ctx)
	defer p.deferTimeTaken(ctx)

	email, ok := p.requireAuth(ctx)
	if !ok {
		return
	}

	if !p.enforceRateLimit(ctx, p.db.RedisRateLimiter, constants.RateLimitGetAllPuzzlesKeyPrefix+email, getAllPuzzlesRateLimit) {
		return
	}

	puzzles, err := p.service.GetAllPuzzlesForUser(ctx, email)
	if err != nil {
		p.writeServiceError(ctx, err)
		return
	}

	p.writeJSONResponse(ctx, http.StatusOK, puzzles)
}

func (p *puzzleHandler) GetPuzzleByID(ctx *fasthttp.RequestCtx) {
	p.timeTakenStart(ctx)
	defer p.deferTimeTaken(ctx)

	email, ok := p.requireAuth(ctx)
	if !ok {
		return
	}

	if !p.enforceRateLimit(ctx, p.db.RedisRateLimiter, constants.RateLimitGetPuzzleByIDKeyPrefix+email, getPuzzleByIDRateLimit) {
		return
	}

	puzzleID := string(ctx.QueryArgs().Peek("puzzle_id"))

	details, err := p.service.GetPuzzleByID(ctx, models.GetPuzzleByIDRequest{
		Email:    email,
		PuzzleID: puzzleID,
	})
	if err != nil {
		p.writeServiceError(ctx, err)
		return
	}

	p.writeJSONResponse(ctx, http.StatusOK, details)
}

func (p *puzzleHandler) AddUserPrompt(ctx *fasthttp.RequestCtx) {
	p.timeTakenStart(ctx)
	defer p.deferTimeTaken(ctx)

	email, ok := p.requireAuth(ctx)
	if !ok {
		return
	}

	if !p.enforceRateLimit(ctx, p.db.RedisRateLimiter, constants.RateLimitAddUserPromptKeyPrefix+email, addUserPromptRateLimit) {
		return
	}

	req, err := bindBodyJSON[addUserPromptRequest](ctx)
	if err != nil {
		p.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	details, err := p.service.AddUserPrompt(ctx, models.AddUserPromptRequest{
		PuzzleID:   req.PuzzleID,
		UserPrompt: req.UserPrompt,
		Email:      email,
	})
	if err != nil {
		p.writeServiceError(ctx, err)
		return
	}

	p.writeJSONResponse(ctx, http.StatusOK, details)
}
