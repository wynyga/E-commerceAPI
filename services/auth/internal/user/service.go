package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/auth"
)

type service struct {
	repo Repository
	key  []byte
	ttl  time.Duration
	// dummyHash is compared against when the email is unknown, so a missing account and a
	// wrong password take the same time to reject.
	dummyHash []byte
}

func NewService(repo Repository, jwtKey []byte, ttl time.Duration) Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	return &service{repo: repo, key: jwtKey, ttl: ttl, dummyHash: dummy}
}

var errBadCredentials = apperr.Unauthorized("invalid email or password")

func normalize(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (s *service) Register(ctx context.Context, p RegisterPayload) (User, error) {
	return s.create(ctx, p.Email, p.Password, auth.RoleCustomer)
}

func (s *service) create(ctx context.Context, email, password, role string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	u := User{Email: normalize(email), PasswordHash: string(hash), Role: role}
	id, err := s.repo.Create(ctx, u)
	if err != nil {
		return User{}, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *service) Login(ctx context.Context, p LoginPayload) (LoginResult, error) {
	u, err := s.repo.FindByEmail(ctx, normalize(p.Email))
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(p.Password))
			return LoginResult{}, errBadCredentials
		}
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(p.Password)) != nil {
		return LoginResult{}, errBadCredentials
	}
	token, exp, err := auth.Sign(s.key, u.ID, u.Role, s.ttl)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: exp, User: u}, nil
}

func (s *service) Profile(ctx context.Context, id uint) (User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) EnsureAdmin(ctx context.Context, email, password string) error {
	if _, err := s.repo.FindByEmail(ctx, normalize(email)); err == nil {
		return nil
	} else if !errors.Is(err, apperr.ErrNotFound) {
		return err
	}
	_, err := s.create(ctx, email, password, auth.RoleAdmin)
	return err
}
