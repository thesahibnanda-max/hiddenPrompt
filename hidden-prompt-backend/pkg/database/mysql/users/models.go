package usersRepo

import (
	"fmt"
	"hidden-prompt-backend/pkg/utils"
	"time"
)

type User struct {
	Email      string        `json:"email"`
	Password   string        `json:"password"`
	IsVerified bool          `json:"is_verified"`
	Metadata   utils.JSONMap `json:"metadata"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

func (u *User) validate() error {
	if u == nil {
		return fmt.Errorf("u is nil")
	}
	email, err := utils.ValidateEmail(u.Email)
	if err != nil {
		return err
	}
	u.Email = email

	if passErr := utils.ValidatePassword(u.Password); passErr != nil {
		return passErr
	}

	if u.Metadata == nil {
		u.Metadata = make(utils.JSONMap)
	}

	return nil
}
