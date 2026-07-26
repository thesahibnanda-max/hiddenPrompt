package user

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"hidden-prompt-backend/pkg/constants"
	"hidden-prompt-backend/pkg/database"
	usersRepo "hidden-prompt-backend/pkg/database/mysql/users"
	customErrors "hidden-prompt-backend/pkg/errors"
	"hidden-prompt-backend/pkg/mail"
	"hidden-prompt-backend/pkg/models"
	"hidden-prompt-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type Service interface {
	SignUp(ctx context.Context, req models.SignUpRequest) (string, error)
	LogIn(ctx context.Context, req models.LoginRequest) (email string, isVerified bool, err error)
	IsVerified(ctx context.Context, email string) (bool, error)
	InitiateVerification(ctx context.Context, email string) error
	DecideVerification(ctx context.Context, email string, otp string) error
}

type service struct {
	db      *database.DatabaseParams
	mailSvc mail.Mailer
}

func NewService(db *database.DatabaseParams, mailSvc mail.Mailer) (Service, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if mailSvc == nil {
		return nil, fmt.Errorf("mailSvc is nil")
	}
	return &service{db: db, mailSvc: mailSvc}, nil
}

func (s *service) SignUp(ctx context.Context, req models.SignUpRequest) (string, error) {
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		return "", &customErrors.EMailIsInvalid{Err: err}
	}
	if err := utils.ValidatePassword(req.Password); err != nil {
		return "", &customErrors.PasswordIsInvalid{Err: err}
	}

	if err := s.db.UsersRepository.CreateUser(ctx, usersRepo.User{
		Email:    email,
		Password: req.Password,
	}); err != nil {
		if errors.Is(err, usersRepo.ErrAlreadyExists) {
			return "", &customErrors.UserAlreadyExists{Err: err}
		}
		return "", err
	}

	return email, nil
}

func (s *service) LogIn(ctx context.Context, req models.LoginRequest) (string, bool, error) {
	email, err := utils.ValidateEmail(req.Email)
	if err != nil {
		return "", false, &customErrors.EMailIsInvalid{Err: err}
	}

	u, err := s.db.UsersRepository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, usersRepo.ErrNoRows) {
			return "", false, &customErrors.InvalidCredentials{Err: err}
		}
		return "", false, err
	}

	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(u.Password)) != 1 {
		return "", false, &customErrors.InvalidCredentials{Err: errors.New("invalid email or password")}
	}

	return email, u.IsVerified, nil
}

func (s *service) IsVerified(ctx context.Context, email string) (bool, error) {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return false, &customErrors.EMailIsInvalid{Err: err}
	}

	u, err := s.db.UsersRepository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, usersRepo.ErrNoRows) {
			return false, &customErrors.UserNotFound{Err: err}
		}
		return false, err
	}

	return u.IsVerified, nil
}

func (s *service) InitiateVerification(ctx context.Context, email string) error {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return &customErrors.EMailIsInvalid{Err: err}
	}

	u, err := s.db.UsersRepository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, usersRepo.ErrNoRows) {
			return &customErrors.UserNotFound{Err: err}
		}
		return err
	}

	if u.IsVerified {
		return &customErrors.UserAlreadyVerified{Err: errors.New("user already verified")}
	}

	otp, err := utils.GenerateRandomString(utils.ChooseOneRandomly(5, uint(6)))
	if err != nil {
		return err
	}

	key := constants.OTPRedisKeyPrefix + email
	if err := s.db.RedisRepository.Set(ctx, key, otp, constants.OTPValidity); err != nil {
		return err
	}

	if err := s.mailSvc.SendOTPMail(ctx, email, otp); err != nil {
		return err
	}

	return nil
}

func (s *service) DecideVerification(ctx context.Context, email string, otp string) error {
	email, err := utils.ValidateEmail(email)
	if err != nil {
		return &customErrors.EMailIsInvalid{Err: err}
	}

	key := constants.OTPRedisKeyPrefix + email
	stored, err := s.db.RedisRepository.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &customErrors.OTPInvalidOrExpired{Err: err}
		}
		return err
	}

	if subtle.ConstantTimeCompare([]byte(otp), []byte(stored)) != 1 {
		return &customErrors.OTPInvalidOrExpired{Err: errors.New("otp mismatch")}
	}

	if err := s.db.UsersRepository.UpdateUserVerificationToTrue(ctx, email); err != nil {
		return err
	}

	_ = s.db.RedisRepository.Delete(ctx, key)

	return nil
}
