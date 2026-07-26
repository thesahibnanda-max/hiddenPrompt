package handler

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/service/user"
	"net/http"

	redisRate "github.com/go-redis/redis_rate/v10"
	"github.com/valyala/fasthttp"
)

// Per-endpoint rate limits, tuned for a resource-constrained (0.1 CPU /
// 256MB) deployment backed entirely by free-tier services: the limiter
// itself (redis_rate's GCRA algorithm) already smooths bursts rather than
// enforcing a hard fixed-window cliff, so these numbers only need to pick
// a sensible sustained rate per endpoint, not fight the algorithm.
var (
	// SignUp/LogIn are keyed by client IP (pre-auth, no email known yet).
	signUpRateLimit = redisRate.PerHour(8)    // one-time action per user; guards against bulk bot signups
	logInRateLimit  = redisRate.PerMinute(10) // generous enough for a few mistyped-password retries, well below brute-force territory

	// InitiateVerification/DecideVerification are keyed by the
	// authenticated user's email (requireAuth already resolved it).
	initiateVerificationRateLimit = redisRate.PerHour(3)   // sends a real email through a free-tier SMTP quota - kept tight
	decideVerificationRateLimit   = redisRate.PerMinute(6) // a few genuine retries, but stops a tight OTP-guessing loop
)

type UserHandler interface {
	SignUp(ctx *fasthttp.RequestCtx)
	LogIn(ctx *fasthttp.RequestCtx)
	InitiateVerification(ctx *fasthttp.RequestCtx)
	DecideVerification(ctx *fasthttp.RequestCtx)
}

type userHandler struct {
	base
	service user.Service
	db      *database.DatabaseParams
}

func NewUserHandler(service user.Service, db *database.DatabaseParams) (UserHandler, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return &userHandler{service: service, db: db}, nil
}

func (u *userHandler) SignUp(ctx *fasthttp.RequestCtx) {
	u.timeTakenStart(ctx)
	defer u.deferTimeTaken(ctx)

	if !u.enforceRateLimit(ctx, u.db.RedisRateLimiter, constants.RateLimitSignUpKeyPrefix+u.clientIP(ctx), signUpRateLimit) {
		return
	}

	req, err := bindBodyJSON[models.SignUpRequest](ctx)
	if err != nil {
		u.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	email, err := u.service.SignUp(ctx, req)
	if err != nil {
		u.writeServiceError(ctx, err)
		return
	}

	if err := u.setAuth(ctx, email); err != nil {
		u.writeUnknownErrorResponse(ctx, err)
		return
	}

	u.writeJSONResponse(ctx, http.StatusCreated, successResponse{Message: "Account created successfully"})
}

func (u *userHandler) LogIn(ctx *fasthttp.RequestCtx) {
	u.timeTakenStart(ctx)
	defer u.deferTimeTaken(ctx)

	if !u.enforceRateLimit(ctx, u.db.RedisRateLimiter, constants.RateLimitLogInKeyPrefix+u.clientIP(ctx), logInRateLimit) {
		return
	}

	req, err := bindBodyJSON[models.LoginRequest](ctx)
	if err != nil {
		u.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	email, isVerified, err := u.service.LogIn(ctx, req)
	if err != nil {
		u.writeServiceError(ctx, err)
		return
	}

	if err := u.setAuth(ctx, email); err != nil {
		u.writeUnknownErrorResponse(ctx, err)
		return
	}

	u.writeJSONResponse(ctx, http.StatusOK, loginResponse{Message: "Logged in successfully", IsVerified: isVerified})
}

func (u *userHandler) InitiateVerification(ctx *fasthttp.RequestCtx) {
	u.timeTakenStart(ctx)
	defer u.deferTimeTaken(ctx)

	email, ok := u.requireAuth(ctx)
	if !ok {
		return
	}

	if !u.enforceRateLimit(ctx, u.db.RedisRateLimiter, constants.RateLimitInitiateVerificationKeyPrefix+email, initiateVerificationRateLimit) {
		return
	}

	if err := u.service.InitiateVerification(ctx, email); err != nil {
		u.writeServiceError(ctx, err)
		return
	}

	u.writeJSONResponse(ctx, http.StatusOK, successResponse{Message: "Verification email sent"})
}

func (u *userHandler) DecideVerification(ctx *fasthttp.RequestCtx) {
	u.timeTakenStart(ctx)
	defer u.deferTimeTaken(ctx)

	email, ok := u.requireAuth(ctx)
	if !ok {
		return
	}

	if !u.enforceRateLimit(ctx, u.db.RedisRateLimiter, constants.RateLimitDecideVerificationKeyPrefix+email, decideVerificationRateLimit) {
		return
	}

	req, err := bindBodyJSON[decideVerificationRequest](ctx)
	if err != nil {
		u.writeErrorResponse(ctx, err.Error(), http.StatusBadRequest, false)
		return
	}

	if err := u.service.DecideVerification(ctx, email, req.OTP); err != nil {
		u.writeServiceError(ctx, err)
		return
	}

	u.writeJSONResponse(ctx, http.StatusOK, successResponse{Message: "Account verified successfully"})
}
