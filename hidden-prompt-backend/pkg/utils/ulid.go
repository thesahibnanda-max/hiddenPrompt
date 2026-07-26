package utils

import "github.com/oklog/ulid/v2"

func GenerateValidULID() string {
	return ulid.Make().String()
}

func IsValidULID(str string) bool {
	_, err := ulid.Parse(str)
	return err == nil
}
