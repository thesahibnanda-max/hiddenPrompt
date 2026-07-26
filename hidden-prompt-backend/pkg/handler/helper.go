package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hidden-prompt-backend/pkg/constants"
	customErrors "hidden-prompt-backend/pkg/errors"
	"hidden-prompt-backend/pkg/service/admin"
	"hidden-prompt-backend/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	goxErrors "github.com/devlibx/gox-base/v2/errors"
	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	redisRate "github.com/go-redis/redis_rate/v10"
	"github.com/valyala/fasthttp"
)

var errUnauthorized = errors.New("Please LogIn or SignUp")

// base holds every request/response helper shared across all four handler
// structs (userHandler, puzzleHandler, adminHandler, healthHandler). None
// of them hold any handler-specific state - they operate purely on the
// *fasthttp.RequestCtx passed in - so base itself is stateless; embedding
// it is Go's idiomatic way to give a group of unrelated structs a shared
// set of methods without inventing a false "is-a" relationship between
// them or forcing them all to share one god-struct's fields.
type base struct{}

func (b base) timeTakenStart(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue(constants.TimeTakenContextKey, time.Now())
}

func (b base) deferTimeTaken(ctx *fasthttp.RequestCtx) {
	timeInterface := ctx.UserValue(constants.TimeTakenContextKey)
	timeTyped, ok := timeInterface.(time.Time)
	if !ok {
		return
	}
	ctx.Response.Header.Set(constants.XTimeTaken, time.Since(timeTyped).String())
}

// bindBodyJSON stays a free function: Go does not allow methods to declare
// their own type parameters, so a generic helper like this one has no
// legal method form - it can only be attached to a generic receiver type,
// which would be the wrong shape here (T varies per call, not per handler
// instance).
func bindBodyJSON[T any](ctx *fasthttp.RequestCtx) (T, error) {
	var obj T
	if err := json.Unmarshal(ctx.Request.Body(), &obj); err != nil {
		return obj, err
	}
	return obj, nil
}

func (b base) handleCustomErrors(err error) (string, int, bool) {
	if _, ok := goxErrors.AsTyped[*customErrors.EMailIsInvalid](err); ok {
		return "Please enter a valid email address.", http.StatusBadRequest, true
	}
	if passwordErr, ok := goxErrors.AsTyped[*customErrors.PasswordIsInvalid](err); ok {
		// The underlying message is already a specific, user-facing reason
		// (e.g. "Password must contain at least one uppercase letter."),
		// crafted by utils.ValidatePassword - show it directly.
		return passwordErr.Error(), http.StatusBadRequest, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.UserAlreadyExists](err); ok {
		return "An account with this email already exists. Try logging in instead.", http.StatusConflict, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.InvalidCredentials](err); ok {
		return "The email or password you entered is incorrect.", http.StatusUnauthorized, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.UserNotFound](err); ok {
		return "We couldn't find an account with that email.", http.StatusNotFound, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.OTPInvalidOrExpired](err); ok {
		return "That verification code is invalid or has expired. Please request a new one.", http.StatusBadRequest, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.UserAlreadyVerified](err); ok {
		return "This account is already verified.", http.StatusConflict, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.UserIsNotVerified](err); ok {
		return "Please verify your email before continuing.", http.StatusForbidden, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.PuzzleIDIsInvalid](err); ok {
		return "That puzzle ID doesn't look right.", http.StatusBadRequest, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.PuzzleNotFound](err); ok {
		return "We couldn't find that puzzle.", http.StatusNotFound, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.PuzzleAlreadyWon](err); ok {
		return "You've already solved this puzzle!", http.StatusConflict, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.UserPromptIsInvalid](err); ok {
		return "Please enter a guess before submitting.", http.StatusBadRequest, true
	}
	if _, ok := goxErrors.AsTyped[*customErrors.MusicDetailsRequired](err); ok {
		return "Please provide at least one music detail to save.", http.StatusBadRequest, true
	}

	return "", 0, false
}

// setAuth stamps the response with an encrypted-email auth token.
func (b base) setAuth(ctx *fasthttp.RequestCtx, email string) error {
	encryptedEmail, err := utils.EncryptString(email)
	if err != nil {
		return err
	}
	ctx.Response.Header.Set(constants.XAuthToken, encryptedEmail)
	return nil
}

// getAuth is setAuth's counterpart: it reads and decrypts the auth token
// off the request. It returns errUnauthorized whenever the token is
// missing, empty, or fails to decrypt.
func (b base) getAuth(ctx *fasthttp.RequestCtx) (string, error) {
	token := string(ctx.Request.Header.Peek(constants.XAuthToken))
	if token == "" {
		return "", errUnauthorized
	}

	email, err := utils.DecryptString(token)
	if err != nil || email == "" {
		return "", errUnauthorized
	}

	return email, nil
}

// requireAuth calls getAuth and, on failure, writes the standard 401
// response itself so protected handlers can just check ok and return.
func (b base) requireAuth(ctx *fasthttp.RequestCtx) (string, bool) {
	email, err := b.getAuth(ctx)
	if err != nil {
		b.writeErrorResponse(ctx, err.Error(), http.StatusUnauthorized, true)
		return "", false
	}
	return email, true
}

// requireAdminAuth checks the X-Admin-Key header against svc and, on
// failure, writes the standard error response itself - same calling
// convention as requireAuth, just gated by a static shared secret instead
// of a per-user token.
func (b base) requireAdminAuth(ctx *fasthttp.RequestCtx, svc admin.Service) bool {
	key := string(ctx.Request.Header.Peek(constants.XAdminKey))
	if !svc.ValidateAdminKey(key) {
		b.writeErrorResponse(ctx, "Invalid or missing admin key.", http.StatusUnauthorized, true)
		return false
	}
	return true
}

func (b base) writeJSONResponse(ctx *fasthttp.RequestCtx, statusCode int, payload any) {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentType(constants.ApplicationJSON)
	_, _ = ctx.Write(goxJsonUtils.ObjectToBytesSuppressError(payload))
}

func (b base) writeErrorResponse(ctx *fasthttp.RequestCtx, message string, statusCode int, showAsIs bool) {
	b.writeJSONResponse(ctx, statusCode, errorResponse{
		ShowResponseAsIs: showAsIs,
		ErrorMessage:     message,
		StatusCode:       statusCode,
	})
}

func (b base) writeUnknownErrorResponse(ctx *fasthttp.RequestCtx, err error) {
	b.writeErrorResponse(ctx, err.Error(), http.StatusInternalServerError, false)
}

// writeServiceError maps a service-layer error to its HTTP response,
// falling back to a generic 500 for anything handleCustomErrors doesn't
// recognize.
func (b base) writeServiceError(ctx *fasthttp.RequestCtx, err error) {
	if message, statusCode, ok := b.handleCustomErrors(err); ok {
		b.writeErrorResponse(ctx, message, statusCode, true)
		return
	}
	b.writeUnknownErrorResponse(ctx, err)
}

// clientIP resolves the caller's address for pre-auth (per-IP) rate
// limiting. It prefers a proxy-forwarded address over the raw connection
// address, since this service is expected to run behind a platform load
// balancer - without this, RemoteIP would resolve to the proxy's own
// address for every request, collapsing per-IP limiting into one shared
// global limit.
func (b base) clientIP(ctx *fasthttp.RequestCtx) string {
	if xff := ctx.Request.Header.Peek("X-Forwarded-For"); len(xff) > 0 {
		if idx := bytes.IndexByte(xff, ','); idx >= 0 {
			xff = xff[:idx]
		}
		if ip := strings.TrimSpace(string(xff)); ip != "" {
			return ip
		}
	}
	return ctx.RemoteIP().String()
}

// enforceRateLimitTimeout bounds limiter.Allow's Redis round-trip -
// defense-in-depth on top of the Redis client's own short connection
// timeouts (pkg/config/redis.go), so a rate-limit check can never block a
// request for long regardless of exactly which underlying timeout
// parameter would otherwise catch a given failure mode. Since a limiter
// error already fails open below, a short timeout here costs nothing.
const enforceRateLimitTimeout = 2 * time.Second

// enforceRateLimit checks key against limit using the shared Redis-backed
// limiter, writing a 429 response and returning false when the caller
// should be rejected. A limiter error (e.g. a transient Redis hiccup on a
// free-tier instance) fails open - the rate limiter is a protective layer,
// not core security, so it shouldn't itself become a single point of
// failure that takes the whole API down.
func (b base) enforceRateLimit(ctx *fasthttp.RequestCtx, limiter *redisRate.Limiter, key string, limit redisRate.Limit) bool {
	if limiter == nil {
		return true
	}

	limitCtx, cancel := context.WithTimeout(ctx, enforceRateLimitTimeout)
	defer cancel()

	result, err := limiter.Allow(limitCtx, key, limit)
	if err != nil {
		return true
	}

	if result.Allowed > 0 {
		return true
	}

	ctx.Response.Header.Set("Retry-After", strconv.Itoa(max(1, int(result.RetryAfter.Seconds()))))
	b.writeErrorResponse(ctx, "Too many requests. Please slow down and try again shortly.", http.StatusTooManyRequests, true)
	return false
}
