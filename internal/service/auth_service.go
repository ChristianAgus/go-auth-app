package service

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"auth-app/internal/model"
	"auth-app/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrUsernameTaken      = errors.New("username sudah dipakai")
	ErrEmailTaken         = errors.New("email sudah terdaftar")
	ErrInvalidToken       = errors.New("token tidak valid")
)

// ValidationError = input dari user salah (nanti jadi HTTP 400)
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

// dummyHash dipakai saat email nggak ditemukan, supaya waktu responsnya
// tetap mirip dengan kasus password salah (cegah user enumeration).
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// <-- BARU: interface UserStore (kontrak yang dibutuhkan service ini)
type UserStore interface {
	Create(ctx context.Context, u *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uint64) (*model.User, error)
}

// <-- BARU: cek saat compile bahwa UserRepository memenuhi UserStore
var _ UserStore = (*repository.UserRepository)(nil)

type AuthService struct {
	users     UserStore // <-- BERUBAH: sebelumnya *repository.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

// <-- BERUBAH: parameter pertama sekarang UserStore
func NewAuthService(users UserStore, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))

	if !usernameRe.MatchString(username) {
		return nil, &ValidationError{"username 3-30 karakter, hanya huruf, angka, dan underscore"}
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, &ValidationError{"format email tidak valid"}
	}
	if len(password) < 8 {
		return nil, &ValidationError{"password minimal 8 karakter"}
	}
	if len(password) > 72 { // batas bcrypt
		return nil, &ValidationError{"password maksimal 72 karakter"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{Username: username, Email: email, PasswordHash: string(hash)}
	if err := s.users.Create(ctx, user); err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateUsername):
			return nil, ErrUsernameTaken
		case errors.Is(err, repository.ErrDuplicateEmail):
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return s.users.GetByID(ctx, user.ID) // ambil ulang biar created_at ikut terisi
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) GetUser(ctx context.Context, id uint64) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	return user, err
}

func (s *AuthService) generateToken(userID uint64) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(userID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// ParseToken memvalidasi token dan mengembalikan user ID di dalamnya.
func (s *AuthService) ParseToken(raw string) (uint64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims,
		func(t *jwt.Token) (any, error) { return s.jwtSecret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}

	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return id, nil
}