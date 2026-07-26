package constants

import "time"

const (
	OTPValidity       = 5 * time.Minute
	OTPRedisKeyPrefix = "otp:verification:"
)
