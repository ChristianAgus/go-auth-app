package repository

import (
	"context"
	"database/sql"
	"errors"

	"auth-app/internal/model"
)

type NoteRepository struct {
	db *sql.DB
}

func NewNoteRepository(db *sql.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) Create(ctx context.Context, n *model.Note) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO notes (user_id, title, content) VALUES (?, ?, ?)`,
		n.UserID, n.Title, n.Content,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	n.ID = uint64(id)
	return nil
}

// GetByID hanya mengembalikan catatan kalau id DAN pemiliknya cocok.
func (r *NoteRepository) GetByID(ctx context.Context, id, userID uint64) (*model.Note, error) {
	var n model.Note
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, content, created_at, updated_at
		 FROM notes WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NoteRepository) List(ctx context.Context, userID uint64, search string, limit, offset int) ([]model.Note, int, error) {
	where := "WHERE user_id = ?"
	args := []any{userID}

	if search != "" {
		where += " AND (title LIKE ? OR content LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notes "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, content, created_at, updated_at FROM notes `+where+
			` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	notes := make([]model.Note, 0) // supaya JSON-nya [] bukan null
	for rows.Next() {
		var n model.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, 0, err
		}
		notes = append(notes, n)
	}
	return notes, total, rows.Err()
}

func (r *NoteRepository) Update(ctx context.Context, n *model.Note) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE notes SET title = ?, content = ? WHERE id = ? AND user_id = ?`,
		n.Title, n.Content, n.ID, n.UserID,
	)
	return err
}

func (r *NoteRepository) Delete(ctx context.Context, id, userID uint64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM notes WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}