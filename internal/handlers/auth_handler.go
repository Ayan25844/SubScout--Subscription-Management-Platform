package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Ayan25844/subscout/internal/database"
	"github.com/Ayan25844/subscout/internal/middleware"
	"github.com/Ayan25844/subscout/internal/models"
	"github.com/Ayan25844/subscout/pkg/auth"
	"github.com/Ayan25844/subscout/pkg/dto"
	"github.com/Ayan25844/subscout/pkg/validator"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *database.Service
}

// @Produce json
// @Tags Currency Routes
// @Router /currencies [get]
// @Summary Get the list of all currencies
// @Failure 404 {string} string "No currencies found"
// @Failure 500 {string} string "Internal server error"
// @Description Currency route to get a list of all currencies
// @Success 200 {array} models.Currency "List of currencies retrieved successfully"
func (h *AuthHandler) GetCurrencies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `SELECT * FROM currencies`
	rows, errQuery := h.DB.Pool.Query(ctx, query)
	if errQuery != nil || rows.Err() != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var currencies []models.Currency
	for rows.Next() {
		var c models.Currency
		if errScan := rows.Scan(&c.ID, &c.Code); errScan != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		currencies = append(currencies, c)
	}
	if len(currencies) == 0 {
		http.Error(w, "No currencies found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currencies)
}

// @Accept json
// @Produce json
// @Tags Public Routes
// @Router /auth/register [post]
// @Summary Register a new user account
// @Failure 400 {string} string "Invalid input"
// @Description Public route to create a user account
// @Param request body dto.User_DTO true "New Account"
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Email id already exists"
// @Success 201 {string} string "User account created successfully"
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.User_DTO
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errEmail := validator.IsValidEmail(req.Email, "Email id", false); errEmail != "" {
		http.Error(w, errEmail, http.StatusBadRequest)
		return
	}
	if errCurrID := validator.ValidateRequiredUUID(req.CurrencyID, "Currency ID", true); errCurrID != "" {
		http.Error(w, errCurrID, http.StatusBadRequest)
		return
	}
	if errPass := validator.ValidatePassword(nil, req.Password, false); errPass != "" {
		http.Error(w, errPass, http.StatusBadRequest)
		return
	}
	hashedPassword, errHash := auth.HashPassword(*req.Password)
	if errHash != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var errQuery error
	var user models.User
	if req.CurrencyID != nil {
		query := `INSERT INTO users (email, password_hash, currency_id) VALUES ($1, $2, $3) RETURNING id, email, currency_id, role, created_at, updated_at`
		errQuery = h.DB.Pool.QueryRow(ctx, query, req.Email, hashedPassword, req.CurrencyID).Scan(&user.ID, &user.Email, &user.CurrencyID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	} else {
		query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, currency_id, role, created_at, updated_at`
		errQuery = h.DB.Pool.QueryRow(ctx, query, req.Email, hashedPassword).Scan(&user.ID, &user.Email, &user.CurrencyID, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	}
	if errQuery != nil {
		var pgErr *pgconn.PgError
		if errors.As(errQuery, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Email id already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// @Accept json
// @Produce json
// @Tags Public Routes
// @Router /auth/login [post]
// @Summary Create a new user login
// @Failure 400 {string} string "Invalid input"
// @Description Public route to log in the user
// @Param request body dto.Login true "New Login"
// @Success 201 {string} string "Login successful"
// @Failure 500 {string} string "Internal server error"
// @Failure 401 {string} string "Unauthorized: Invalid email or password"
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.Login
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errEmail := validator.IsValidEmail(req.Email, "Email id", false); errEmail != "" {
		http.Error(w, errEmail, http.StatusBadRequest)
		return
	}
	if errPassword := validator.ValidateRequiredString(req.Password, "Password", false); errPassword != "" {
		http.Error(w, errPassword, http.StatusBadRequest)
		return
	}
	var id uuid.UUID
	var dbHash, role string
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `SELECT id, password_hash, role FROM users WHERE email = $1`
	errQuery := h.DB.Pool.QueryRow(ctx, query, req.Email).Scan(&id, &dbHash, &role)
	if errQuery != nil {
		http.Error(w, "Invalid email or password", http.StatusInternalServerError)
		return
	}
	if !auth.CheckPasswordHash(*req.Password, dbHash) {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}
	token, errToken := auth.GenerateToken(id, role)
	if errToken != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   token,
		"message": "Login successful",
	})
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /users/me [put]
// @Summary Update the profile of a user
// @Failure 400 {string} string "Invalid input"
// @Description User route to update a user profile
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Email id already exists"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Success 200 {object} models.User "User profile updated successfully"
// @Param request body dto.Update_User_Profile true "Updated user profile"
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	var req dto.Update_User_Profile
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errEmail := validator.IsValidEmail(req.Email, "Email id", true); errEmail != "" {
		http.Error(w, errEmail, http.StatusBadRequest)
		return
	}
	if errCurrID := validator.ValidateRequiredUUID(req.CurrencyID, "Currency ID", true); errCurrID != "" {
		http.Error(w, errCurrID, http.StatusBadRequest)
		return
	}
	var profile models.User
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `
		UPDATE users 
		SET currency_id = COALESCE($1, currency_id), email = COALESCE($2, email), updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id, role, currency_id, email, created_at, updated_at`
	errQuery := h.DB.Pool.QueryRow(ctx, query, req.CurrencyID, req.Email, claims.UserID).
		Scan(&profile.ID, &profile.Role, &profile.CurrencyID, &profile.Email, &profile.CreatedAt, &profile.UpdatedAt)
	if errQuery != nil {
		var pgErr *pgconn.PgError
		if errors.As(errQuery, &pgErr) {
			if pgErr.Code == "23505" {
				http.Error(w, "Email ID already exists", http.StatusConflict)
				return
			}
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /users/me/password [put]
// @Summary Update the password of a user
// @Failure 400 {string} string "Invalid input"
// @Description User route to update a user password
// @Failure 500 {string} string "Internal server error"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Success 200 {string} string "User password updated successfully"
// @Param request body dto.Update_User_Password true "Updated user password"
func (h *AuthHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	var req dto.Update_User_Password
	if errDecode := json.NewDecoder(r.Body).Decode(&req); errDecode != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errPass := validator.ValidatePassword(req.OldPassword, req.NewPassword, true); errPass != "" {
		http.Error(w, errPass, http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var currentHash string
	errQuery := h.DB.Pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", claims.UserID).Scan(&currentHash)
	if errQuery != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if errCompare := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(*req.OldPassword)); errCompare != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	newHash, errHash := bcrypt.GenerateFromPassword([]byte(*req.NewPassword), bcrypt.DefaultCost)
	if errHash != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	_, errQuery = h.DB.Pool.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		newHash, claims.UserID)
	if errQuery != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User Password updated successfully"})
}

// @Security Bearer
// @Tags User Routes
// @Success 204 "No Content"
// @Summary Delete a user account
// @Router /users/me/delete [delete]
// @Failure 500 {string} string "Internal server error"
// @Failure 400 {string} string "No user account found"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Description User route to delete a user account for the logged-in user
func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `DELETE FROM users WHERE id = $1`
	commandTag, errQuery := h.DB.Pool.Exec(ctx, query, claims.UserID)
	if errQuery != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "No user account found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
