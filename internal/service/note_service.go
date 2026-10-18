package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"auth-app/internal/model"
	"auth-app/internal/repository"
)

var ErrNoteNotFound = errors.New("catatan tidak ditemukan")

type NoteService struct {
	notes *repository.NoteRepository
}

func NewNoteService(notes *repository.NoteRepository) *NoteService {
	return &NoteService{notes: notes}
}

func validateNote(title, content string) (string, string, error) {
	title = strings.TrimSpace(title)

	if n := utf8.RuneCountInString(title); n < 1 || n > 200 {
		return "", "", &ValidationError{"judul wajib diisi, maksimal 200 karakter"}
	}
	if utf8.RuneCountInString(content) > 10000 {
		return "", "", &ValidationError{"isi catatan maksimal 10.000 karakter"}
	}
	return title, content, nil
}

func (s *NoteService) Create(ctx context.Context, userID uint64, title, content string) (*model.Note, error) {
	title, content, err := validateNote(title, content)
	if err != nil {
		return nil, err
	}

	note := &model.Note{UserID: userID, Title: title, Content: content}
	if err := s.notes.Create(ctx, note); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, note.ID) // ambil ulang biar created_at ikut terisi
}

func (s *NoteService) Get(ctx context.Context, userID, id uint64) (*model.Note, error) {
	note, err := s.notes.GetByID(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoteNotFound
	}
	return note, err
}

func (s *NoteService) List(ctx context.Context, userID uint64, search string, page, limit int) ([]model.Note, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	return s.notes.List(ctx, userID, strings.TrimSpace(search), limit, (page-1)*limit)
}

func (s *NoteService) Update(ctx context.Context, userID, id uint64, title, content string) (*model.Note, error) {
	title, content, err := validateNote(title, content)
	if err != nil {
		return nil, err
	}

	// cek kepemilikan dulu; kalau bukan milik user ini -> ErrNoteNotFound
	note, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	note.Title = title
	note.Content = content
	if err := s.notes.Update(ctx, note); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

func (s *NoteService) Delete(ctx context.Context, userID, id uint64) error {
	err := s.notes.Delete(ctx, id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNoteNotFound
	}
	return err
}