package service

import (
	"context"
	"fmt"

	"cerberus/config"
	"cerberus/ent"
	"cerberus/internal/auth"
	"cerberus/internal/repository"
	apperrors "cerberus/pkg/errors"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

type RegisterInput struct {
	Email      string
	Name       string
	Password   string
	Department string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	Token string
	User  interface{} // Returns *ent.User — typed in the resolver
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (string, *ent.User, error) {
	// Validate input
	if len(input.Password) < 8 {
		return "", nil, apperrors.ValidationError("password must be at least 8 characters")
	}

	// Check if email already exists
	existing, err := s.userRepo.FindByEmail(ctx, input.Email)
	//fmt.Printf("\n\n 10. Checking if email already exists: %s", existing.Email)
	if err != nil {
		return "", nil, apperrors.Internal("failed to check email", err)
	}
	if existing != nil {
		return "", nil, apperrors.Conflict("email already registered")
	}

	// Hash password — never store plaintext
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, apperrors.Internal("failed to hash password", err)
	}

	user, err := s.userRepo.Create(ctx, input.Email, input.Name, string(hash), input.Department)
	if err != nil {
		return "", nil, apperrors.Internal("failed to create user", err)
	}

	token, err := auth.GenerateToken(
		fmt.Sprintf("%d", user.ID),
		user.Email,
		string(user.Role),
		s.cfg.JWT.Secret,
		s.cfg.JWT.ExpiryHours,
	)
	if err != nil {
		return "", nil, apperrors.Internal("failed to generate token", err)
	}

	fmt.Println("12. Generating token")
	fmt.Printf("\n\n Generated token for user %s: %s", user.Email, token)
	return token, user, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (string, *ent.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	fmt.Printf("\n\n 14. -->  Found user in login: %v", user)
	if err != nil {
		return "", nil, apperrors.Internal("login failed", err)
	}
	// Return same error for invalid email and invalid password — prevents user enumeration
	if user == nil {
		return "", nil, apperrors.Unauthorized("invalid credentials")
	}

	if !user.IsActive {
		return "", nil, apperrors.Unauthorized("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		fmt.Printf("\n\n 15. -->  Password mismatch for user: %v", user.Email)
		return "", nil, apperrors.Unauthorized("invalid credentials")
	}

	token, err := auth.GenerateToken(
		fmt.Sprintf("%d", user.ID),
		user.Email,
		string(user.Role),
		s.cfg.JWT.Secret,
		s.cfg.JWT.ExpiryHours,
	)
	if err != nil {
		return "", nil, apperrors.Internal("failed to generate token", err)
	}
	fmt.Printf("16. Generating token for login is: %v", token)
	fmt.Printf("\n\n 16. -->  Generated token for user: %v", user.Email)

	return token, user, nil
}
