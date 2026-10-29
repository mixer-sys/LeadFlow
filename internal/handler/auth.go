package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"leadflow/internal/domain"
	"leadflow/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerRequest struct {
	OrganizationName string `json:"organization_name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func isFormRequest(r *http.Request) bool {
	return strings.Contains(
		r.Header.Get("Content-Type"),
		"application/x-www-form-urlencoded",
	)
}

func (h *AuthHandler) Register(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeRegisterRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.svc.Register(r.Context(), service.RegisterInput{
		OrganizationName: req.OrganizationName,
		Email:            req.Email,
		Password:         req.Password,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if err := setSession(w, r, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create session",
		})
		return
	}

	if isFormRequest(r) {
		http.Redirect(w, r, "/admin/leads", http.StatusSeeOther)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":              user.ID,
		"organization_id": user.OrganizationID,
		"email":           user.Email,
		"role":            user.Role,
	})
}

func (h *AuthHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := decodeLoginRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid email or password",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "login failed",
		})
		return
	}

	if err := setSession(w, r, user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create session",
		})
		return
	}

	if isFormRequest(r) {
		http.Redirect(w, r, "/admin/leads", http.StatusSeeOther)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              user.ID,
		"organization_id": user.OrganizationID,
		"email":           user.Email,
		"role":            user.Role,
	})
}

func (h *AuthHandler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := getUserIDFromSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":              user.ID,
		"organization_id": user.OrganizationID,
		"email":           user.Email,
		"role":            user.Role,
	})
}

func (h *AuthHandler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	if err := clearSession(w, r); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "logout failed",
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func decodeRegisterRequest(
	r *http.Request,
) (registerRequest, error) {
	var req registerRequest

	if isFormRequest(r) {
		if err := r.ParseForm(); err != nil {
			return req, err
		}

		req.OrganizationName = r.FormValue("organization_name")
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")

		return req, nil
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}

	return req, nil
}

func decodeLoginRequest(
	r *http.Request,
) (loginRequest, error) {
	var req loginRequest

	if isFormRequest(r) {
		if err := r.ParseForm(); err != nil {
			return req, err
		}

		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")

		return req, nil
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}

	return req, nil
}
