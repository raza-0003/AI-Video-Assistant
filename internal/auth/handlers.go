package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	DB           *pgxpool.Pool
	JWTSecret    string
	JWTExpiryHrs int
}

func NewHandler(db *pgxpool.Pool, secret string, expiryHrs int) *Handler {
	return &Handler{DB: db, JWTSecret: secret, JWTExpiryHrs: expiryHrs}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		FullName string `json:"full_name"`
	} `json:"user"`
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerRequest true "Registration payload"
// @Success      201 {object} authResponse
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		httpError(w, http.StatusBadRequest, "email required, password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	id := uuid.NewString()
	ctx := r.Context()
	_, err = h.DB.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name) VALUES ($1, $2, $3, $4)`,
		id, req.Email, string(hash), req.FullName,
	)
	if err != nil {
		httpError(w, http.StatusConflict, "email already registered")
		return
	}

	token, err := GenerateToken(h.JWTSecret, id, req.Email, h.JWTExpiryHrs)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = id
	resp.User.Email = req.Email
	resp.User.FullName = req.FullName

	writeJSON(w, http.StatusCreated, resp)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login godoc
// @Summary      Log in and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Login payload"
// @Success      200 {object} authResponse
// @Router       /api/v1/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var id, hash, fullName string
	ctx := r.Context()
	err := h.DB.QueryRow(ctx,
		`SELECT id, password_hash, full_name FROM users WHERE email = $1`, req.Email,
	).Scan(&id, &hash, &fullName)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		httpError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := GenerateToken(h.JWTSecret, id, req.Email, h.JWTExpiryHrs)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = id
	resp.User.Email = req.Email
	resp.User.FullName = fullName

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

var _ = context.Background
