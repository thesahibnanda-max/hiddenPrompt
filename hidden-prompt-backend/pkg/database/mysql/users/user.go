package usersRepo

import (
	"context"
	"database/sql"
	"errors"
	usersInternalMySQL "hidden-prompt-backend/pkg/database/mysql/users/internal"
	"hidden-prompt-backend/pkg/utils"

	goxErrors "github.com/devlibx/gox-base/v2/errors"
	"github.com/go-sql-driver/mysql"
)

var (
	ErrNoRows        = errors.New("no user rows available")
	ErrAlreadyExists = errors.New("user already exists")
)

type Repository interface {
	CreateUser(ctx context.Context, u User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUserVerificationToTrue(ctx context.Context, email string) error
	DeleteUserByEmail(ctx context.Context, email string) error
}

type repo struct {
	q usersInternalMySQL.Querier
}

func (r *repo) CreateUser(ctx context.Context, u User) error {
	if valErr := u.validate(); valErr != nil {
		return valErr
	}

	passwordHash, err := utils.EncryptString(u.Password)
	if err != nil {
		return err
	}

	if err := r.q.CreateUser(ctx, usersInternalMySQL.CreateUserParams{
		UserEmail:    u.Email,
		PasswordHash: passwordHash,
		IsVerified:   u.IsVerified,
		Metadata:     u.Metadata,
	}); err != nil {
		if mysqlErr, ok := goxErrors.AsTyped[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
			return ErrAlreadyExists
		}
		return err
	}

	return nil
}

func (r *repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return r.getUserByEmail(ctx, email)
}

func (r *repo) UpdateUserVerificationToTrue(ctx context.Context, email string) error {
	u, err := r.getUserByEmail(ctx, email)
	if err != nil {
		return err
	}

	return r.q.UpdateUserVerificationToTrue(ctx, u.Email)
}

func (r *repo) DeleteUserByEmail(ctx context.Context, email string) error {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return err
	}

	return r.q.DeleteUserByEmail(ctx, email)
}

func (r *repo) getUserByEmail(ctx context.Context, email string) (*User, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return nil, err
	}

	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, err
	}

	return r.toModel(u)
}
