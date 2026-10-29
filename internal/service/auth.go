package service

import (
	"context"
	"errors"
	"strings"

	"leadflow/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type AuthRepo interface {
	CreateWithOrganization(
		ctx context.Context,
		organizationName string,
		email string,
		passwordHash string,
	) (*domain.User, error)

	GetByEmail(
		ctx context.Context,
		email string,
	) (*domain.User, error)

	GetByID(
		ctx context.Context,
		id int64,
	) (*domain.User, error)
}

type AuthService struct {
	repo AuthRepo
}

func NewAuthService(repo AuthRepo) *AuthService {
	return &AuthService{repo: repo}
}

type RegisterInput struct {
	OrganizationName string
	Email            string
	Password         string
}

func (s *AuthService) Register(
	ctx context.Context,
	input RegisterInput,
) (*domain.User, error) {
	input.OrganizationName = strings.TrimSpace(input.OrganizationName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.OrganizationName == "" {
		return nil, errors.New("organization name is required")
	}

	if input.Email == "" {
		return nil, errors.New("email is required")
	}

	if len(input.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateWithOrganization(
		ctx,
		input.OrganizationName,
		input.Email,
		string(passwordHash),
	)
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}

		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) GetUserByID(
	ctx context.Context,
	id int64,

) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}
