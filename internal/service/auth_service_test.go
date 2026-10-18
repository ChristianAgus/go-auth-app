package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"auth-app/internal/model"
	"auth-app/internal/repository"
)

const testSecret = "test-secret-test-secret-test-secret-1234"

// fakeUserStore = "database palsu" di memori, jadi test nggak butuh MySQL.
type fakeUserStore struct {
	users  map[uint64]*model.User
	nextID uint64
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: map[uint64]*model.User{}, nextID: 1}
}

func (f *fakeUserStore) Create(_ context.Context, u *model.User) error {
	for _, existing := range f.users {
		if existing.Username == u.Username {
			return repository.ErrDuplicateUsername
		}
		if existing.Email == u.Email {
			return repository.ErrDuplicateEmail
		}
	}
	u.ID = f.nextID
	f.nextID++
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now

	stored := *u
	f.users[u.ID] = &stored
	return nil
}

func (f *fakeUserStore) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			c := *u
			return &c, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserStore) GetByID(_ context.Context, id uint64) (*model.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	c := *u
	return &c, nil
}

func newTestAuthService() *AuthService {
	return NewAuthService(newFakeUserStore(), testSecret, time.Hour)
}

func TestRegister_Validation(t *testing.T) {
	svc := newTestAuthService()

	tests := []struct {
		name     string
		username string
		email    string
		password string
	}{
		{"username terlalu pendek", "ab", "a@mail.com", "password123"},
		{"username ada spasi", "bad name", "a@mail.com", "password123"},
		{"username ada simbol", "bad-name!", "a@mail.com", "password123"},
		{"email tidak valid", "christian", "bukan-email", "password123"},
		{"email kosong", "christian", "", "password123"},
		{"password terlalu pendek", "christian", "a@mail.com", "1234567"},
		{"password lebih dari 72 karakter", "christian", "a@mail.com", strings.Repeat("a", 73)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.username, tt.email, tt.password)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("harusnya ValidationError, dapat: %v", err)
			}
		})
	}
}

func TestRegister_Success(t *testing.T) {
	svc := newTestAuthService()

	user, err := svc.Register(context.Background(), "  Christian ", " CHRIS@Mail.COM ", "rahasia123")
	if err != nil {
		t.Fatalf("register gagal: %v", err)
	}

	if user.Username != "Christian" {
		t.Errorf("username harusnya di-trim, dapat %q", user.Username)
	}
	if user.Email != "chris@mail.com" {
		t.Errorf("email harusnya lowercase dan di-trim, dapat %q", user.Email)
	}
	if user.PasswordHash == "rahasia123" {
		t.Fatal("password tersimpan polos, harusnya di-hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("rahasia123")); err != nil {
		t.Errorf("hash tidak cocok dengan password asli: %v", err)
	}
}

func TestRegister_Duplicate(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "christian", "chris@mail.com", "rahasia123"); err != nil {
		t.Fatalf("register pertama gagal: %v", err)
	}

	_, err := svc.Register(ctx, "christian", "lain@mail.com", "rahasia123")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("harusnya ErrUsernameTaken, dapat: %v", err)
	}

	_, err = svc.Register(ctx, "budi", "chris@mail.com", "rahasia123")
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("harusnya ErrEmailTaken, dapat: %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	registered, err := svc.Register(ctx, "christian", "chris@mail.com", "rahasia123")
	if err != nil {
		t.Fatal(err)
	}

	// email dengan huruf besar + spasi tetap harus bisa login
	token, user, err := svc.Login(ctx, " Chris@Mail.com ", "rahasia123")
	if err != nil {
		t.Fatalf("login gagal: %v", err)
	}
	if token == "" {
		t.Fatal("token kosong")
	}
	if user.ID != registered.ID {
		t.Errorf("user ID salah: %d", user.ID)
	}

	id, err := svc.ParseToken(token)
	if err != nil || id != registered.ID {
		t.Errorf("token harusnya berisi user ID %d, dapat %d (err: %v)", registered.ID, id, err)
	}
}

func TestLogin_Failures(t *testing.T) {
	svc := newTestAuthService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "christian", "chris@mail.com", "rahasia123"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"password salah", "chris@mail.com", "salahbanget"},
		{"email tidak terdaftar", "hantu@mail.com", "rahasia123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, user, err := svc.Login(ctx, tt.email, tt.password)

			// dua-duanya harus menghasilkan error yang sama persis
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("harusnya ErrInvalidCredentials, dapat: %v", err)
			}
			if token != "" || user != nil {
				t.Error("token/user harus kosong kalau login gagal")
			}
		})
	}
}

func signedToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseToken(t *testing.T) {
	svc := newTestAuthService()
	now := time.Now()

	valid := jwt.RegisteredClaims{Subject: "7", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}
	expired := jwt.RegisteredClaims{Subject: "7", ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute))}
	badSubject := jwt.RegisteredClaims{Subject: "abc", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"token valid", signedToken(t, jwt.SigningMethodHS256, []byte(testSecret), valid), false},
		{"token expired", signedToken(t, jwt.SigningMethodHS256, []byte(testSecret), expired), true},
		{"secret berbeda", signedToken(t, jwt.SigningMethodHS256, []byte("secret-lain-yang-sama-sekali-berbeda"), valid), true},
		{"algoritma none (serangan klasik)", signedToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid), true},
		{"subject bukan angka", signedToken(t, jwt.SigningMethodHS256, []byte(testSecret), badSubject), true},
		{"token sampah", "bukan.token.sama.sekali", true},
		{"token kosong", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := svc.ParseToken(tt.token)

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("harusnya ErrInvalidToken, dapat: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("harusnya sukses, dapat: %v", err)
			}
			if id != 7 {
				t.Errorf("user ID harusnya 7, dapat %d", id)
			}
		})
	}
}

func TestGetUser_UnknownID(t *testing.T) {
	svc := newTestAuthService()

	_, err := svc.GetUser(context.Background(), 999)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("harusnya ErrInvalidToken, dapat: %v", err)
	}
}