package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"

	"auth-app/internal/model"
)

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrDuplicateUsername = errors.New("username duplikat")
	ErrDuplicateEmail    = errors.New("email duplikat")
)

const mysqlDuplicateEntry = 1062

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		u.Username, u.Email, u.PasswordHash,
	)
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == mysqlDuplicateEntry {
			// pesan: "Duplicate entry 'x' for key 'users.email'", cek bagian setelah "for key"
			if i := strings.Index(me.Message, "for key"); i >= 0 {
				key := me.Message[i:]
				switch {
				case strings.Contains(key, "username"):
					return ErrDuplicateUsername
				case strings.Contains(key, "email"):
					return ErrDuplicateEmail
				}
			}
		}
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = uint64(id)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.getOne(ctx, `WHERE email = ?`, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	return r.getOne(ctx, `WHERE id = ?`, id)
}

func (r *UserRepository) getOne(ctx context.Context, where string, arg any) (*model.User, error) {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users ` + where

	var u model.User
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}