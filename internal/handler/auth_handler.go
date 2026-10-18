package handler

import (
	"errors"
	"log"
	"net/http"

	"auth-app/internal/middleware"
	"auth-app/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	user, err := h.auth.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	token, user, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": user})
}

func (h *AuthHandler) handleError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError

	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Message)
	case errors.Is(err, service.ErrUsernameTaken), errors.Is(err, service.ErrEmailTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		log.Printf("error tidak terduga: %v", err) // detail cuma di log, bukan ke client
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}