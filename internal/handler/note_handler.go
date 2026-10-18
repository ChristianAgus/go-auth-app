package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"auth-app/internal/middleware"
	"auth-app/internal/service"
)

type NoteHandler struct {
	notes *service.NoteService
}

func NewNoteHandler(notes *service.NoteService) *NoteHandler {
	return &NoteHandler{notes: notes}
}

type noteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var req noteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	note, err := h.notes.Create(r.Context(), userID, req.Title, req.Content)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": note})
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	notes, total, err := h.notes.List(r.Context(), userID, q.Get("q"), page, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": notes,
		"meta": map[string]int{"total": total},
	})
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := noteID(w, r)
	if !ok {
		return
	}

	note, err := h.notes.Get(r.Context(), userID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": note})
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := noteID(w, r)
	if !ok {
		return
	}

	var req noteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	note, err := h.notes.Update(r.Context(), userID, id, req.Title, req.Content)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": note})
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := noteID(w, r)
	if !ok {
		return
	}

	if err := h.notes.Delete(r.Context(), userID, id); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NoteHandler) handleError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError

	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, service.ErrNoteNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("error tidak terduga: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func currentUserID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
	}
	return id, ok
}

// noteID membaca {id} dari URL, contoh: /api/notes/5
func noteID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "catatan tidak ditemukan")
		return 0, false
	}
	return id, true
}