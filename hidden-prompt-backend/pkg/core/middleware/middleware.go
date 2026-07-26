package middleware

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"log/slog"
	"net/http"
	"runtime/debug"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	"github.com/valyala/fasthttp"
)

type panicResponse struct {
	Panic string `json:"panic"`
}

type Middleware interface {
	// AllowCORS wraps next with permissive CORS headers - all origins, all
	// methods, and all headers (including the auth token header) are
	// allowed. Preflight OPTIONS requests are answered directly and never
	// reach next.
	AllowCORS(next fasthttp.RequestHandler) fasthttp.RequestHandler

	// PanicRecovery wraps next so a panic inside it is recovered, logged
	// with a stack trace, and turned into a clean {"panic": "..."} 500
	// response instead of crashing the whole server process.
	PanicRecovery(next fasthttp.RequestHandler) fasthttp.RequestHandler
}

type middleware struct{}

func NewMiddleware() Middleware {
	return &middleware{}
}

func (m *middleware) AllowCORS(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// Access-Control-Allow-Headers only governs what the browser may
		// SEND. Custom response headers (X-Auth-Token, X-Time-Taken) are
		// otherwise invisible to JS on a cross-origin response - fetch's
		// Response.headers.get() silently returns null for them - unless
		// explicitly exposed here. Without this, browser clients can
		// never read the auth token off signup/login responses even
		// though curl and other non-browser clients see it fine, since
		// CORS is a browser-enforced restriction only.
		ctx.Response.Header.Set("Access-Control-Expose-Headers", constants.XAuthToken+", "+constants.XTimeTaken)

		// Echo back whatever headers the browser is asking to send
		// (falling back to "*") so any header - including our custom
		// X-Auth-Token - is always allowed, without needing to enumerate
		// them by name.
		if reqHeaders := ctx.Request.Header.Peek("Access-Control-Request-Headers"); len(reqHeaders) > 0 {
			ctx.Response.Header.SetBytesV("Access-Control-Allow-Headers", reqHeaders)
		} else {
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "*")
		}

		if string(ctx.Method()) == http.MethodOptions {
			// Preflight check only - cache it for a day so browsers don't
			// re-probe on every real request, which matters on a
			// resource-constrained deployment.
			ctx.Response.Header.Set("Access-Control-Max-Age", "86400")
			ctx.SetStatusCode(http.StatusNoContent)
			return
		}

		next(ctx)
	}
}

func (m *middleware) PanicRecovery(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}

			slog.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("path", string(ctx.Path())),
				slog.String("stack", string(debug.Stack())),
			)

			// The handler may have already written partial output before
			// panicking - clear it so the recovery response isn't mixed
			// with garbage from the failed attempt.
			ctx.Response.Reset()
			ctx.SetStatusCode(http.StatusInternalServerError)
			ctx.SetContentType(constants.ApplicationJSON)
			_, _ = ctx.Write(goxJsonUtils.ObjectToBytesSuppressError(panicResponse{
				Panic: fmt.Sprintf("%v", r),
			}))
		}()

		next(ctx)
	}
}
