package handler

type errorResponse struct {
	ShowResponseAsIs bool   `json:"show_response_as_is"`
	ErrorMessage     string `json:"error_message"`
	StatusCode       int    `json:"status_code"`
}

type successResponse struct {
	Message string `json:"message"`
}

type loginResponse struct {
	Message    string `json:"message"`
	IsVerified bool   `json:"is_verified"`
}

type decideVerificationRequest struct {
	OTP string `json:"otp"`
}

type addUserPromptRequest struct {
	PuzzleID   string `json:"puzzle_id"`
	UserPrompt string `json:"user_prompt"`
}

type healthCheckResponse struct {
	Status string            `json:"status"`
	Errors map[string]string `json:"errors,omitempty"`
}
