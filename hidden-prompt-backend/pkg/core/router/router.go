package router

import (
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/core/middleware"
	"hidden-prompt-backend/pkg/handler"
	"net/http"

	goxJsonUtils "github.com/devlibx/gox-base/v2/serialization/utils/json"
	fastrouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

const (
	puzzleBasePath = "/puzzle"
	usersBasPath   = "/users"
	adminBasePath  = "/admin"
	musicBasePath  = "/music"
)

type errorBody struct {
	ErrorMessage string `json:"error_message"`
}

// Router exposes the single fasthttp.RequestHandler entry point a
// fasthttp.Server registers as its Handler.
type Router interface {
	Handler() fasthttp.RequestHandler
}

type router struct {
	handler fasthttp.RequestHandler
}

func NewRouter(
	userHandler handler.UserHandler,
	puzzleHandler handler.PuzzleHandler,
	healthHandler handler.HealthHandler,
	adminHandler handler.AdminHandler,
	musicHandler handler.MusicHandler,
	mw middleware.Middleware,
) (Router, error) {
	if userHandler == nil {
		return nil, fmt.Errorf("userHandler is nil")
	}
	if puzzleHandler == nil {
		return nil, fmt.Errorf("puzzleHandler is nil")
	}
	if healthHandler == nil {
		return nil, fmt.Errorf("healthHandler is nil")
	}
	if adminHandler == nil {
		return nil, fmt.Errorf("adminHandler is nil")
	}
	if musicHandler == nil {
		return nil, fmt.Errorf("musicHandler is nil")
	}
	if mw == nil {
		return nil, fmt.Errorf("mw is nil")
	}

	ew := errorWriter{}

	fr := fastrouter.New()
	fr.RedirectTrailingSlash = true
	fr.RedirectFixedPath = true
	fr.HandleMethodNotAllowed = true
	fr.NotFound = ew.writeNotFound
	fr.MethodNotAllowed = ew.writeMethodNotAllowed

	fr.GET("/health", healthHandler.GetHealth)

	users := fr.Group(usersBasPath)
	users.POST("/signup", userHandler.SignUp)
	users.POST("/login", userHandler.LogIn)
	users.POST("/verify/initiate", userHandler.InitiateVerification)
	users.POST("/verify/decide", userHandler.DecideVerification)

	fr.POST(puzzleBasePath, puzzleHandler.CreatePuzzleForUser)
	fr.GET(puzzleBasePath, puzzleHandler.GetAllPuzzlesForUser)
	puzzles := fr.Group(puzzleBasePath)
	puzzles.GET("/detail", puzzleHandler.GetPuzzleByID)
	puzzles.POST("/guess", puzzleHandler.AddUserPrompt)

	adminGroup := fr.Group(adminBasePath)
	adminGroup.POST("/cache/clear", adminHandler.ClearCache)
	adminGroup.POST("/user/delete", adminHandler.DeleteUser)

	fr.POST(musicBasePath, musicHandler.InsertMusicDetails)
	fr.GET(musicBasePath, musicHandler.GetAllMusicDetails)

	// PanicRecovery is the outermost layer so nothing - including a panic
	// inside CORS handling itself - can escape uncaught and crash the
	// process. fasthttp/router's own OPTIONS handling is left disabled
	// since AllowCORS already fully owns preflight requests upstream.
	return &router{
		handler: mw.PanicRecovery(mw.AllowCORS(fr.Handler)),
	}, nil
}

func (r *router) Handler() fasthttp.RequestHandler {
	return r.handler
}

// errorWriter is stateless - it exists purely to give these three related
// responses a shared receiver instead of three unrelated free functions.
// It's constructed before fr (the fasthttp/router instance) itself, since
// fr.NotFound/fr.MethodNotAllowed need method values to assign at that
// point in NewRouter, ahead of when the router struct is built.
type errorWriter struct{}

func (ew errorWriter) writeNotFound(ctx *fasthttp.RequestCtx) {
	ew.writeJSONError(ctx, http.StatusNotFound, "Not Found")
}

func (ew errorWriter) writeMethodNotAllowed(ctx *fasthttp.RequestCtx) {
	ew.writeJSONError(ctx, http.StatusMethodNotAllowed, "Method Not Allowed")
}

func (ew errorWriter) writeJSONError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentType(constants.ApplicationJSON)
	_, _ = ctx.Write(goxJsonUtils.ObjectToBytesSuppressError(errorBody{ErrorMessage: message}))
}
