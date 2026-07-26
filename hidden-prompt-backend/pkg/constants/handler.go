package constants

const (
	ApplicationJSON     = "application/json"
	XAuthToken          = "X-Auth-Token"
	XAdminKey           = "X-Admin-Key"
	XTimeTaken          = "X-Time-Taken"
	TimeTakenContextKey = "__time_taken__"

	RateLimitSignUpKeyPrefix               = "ratelimit:signup:"
	RateLimitLogInKeyPrefix                = "ratelimit:login:"
	RateLimitInitiateVerificationKeyPrefix = "ratelimit:initiate_verification:"
	RateLimitDecideVerificationKeyPrefix   = "ratelimit:decide_verification:"

	RateLimitCreatePuzzleKeyPrefix  = "ratelimit:create_puzzle:"
	RateLimitGetAllPuzzlesKeyPrefix = "ratelimit:get_all_puzzles:"
	RateLimitGetPuzzleByIDKeyPrefix = "ratelimit:get_puzzle_by_id:"
	RateLimitAddUserPromptKeyPrefix = "ratelimit:add_user_prompt:"

	// RateLimitAdminKeyPrefix is per-IP (pre-auth style) since admin calls
	// have no user identity - it exists to slow down brute-forcing the
	// admin key, not to protect a legitimate high-volume caller.
	RateLimitAdminKeyPrefix = "ratelimit:admin:"

	// RateLimitInsertMusicKeyPrefix is per-IP, same reasoning as
	// RateLimitAdminKeyPrefix (admin-key-gated, no user identity).
	RateLimitInsertMusicKeyPrefix = "ratelimit:insert_music:"
	// RateLimitGetAllMusicKeyPrefix is per-IP since this endpoint has no
	// auth at all.
	RateLimitGetAllMusicKeyPrefix = "ratelimit:get_all_music:"
)
