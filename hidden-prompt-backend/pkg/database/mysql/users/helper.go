package usersRepo

import (
	usersInternalMySQL "hidden-prompt-backend/pkg/database/mysql/users/internal"
	"hidden-prompt-backend/pkg/utils"
)

func (r *repo) toModel(u usersInternalMySQL.User) (*User, error) {

	if u.Metadata == nil {
		u.Metadata = make(utils.JSONMap)
	}

	decryptedPassword, err := utils.DecryptString(u.PasswordHash)
	if err != nil {
		return nil, err
	}

	return &User{
		Email:      u.UserEmail,
		Password:   decryptedPassword,
		IsVerified: u.IsVerified,
		Metadata:   u.Metadata,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}, nil
}
