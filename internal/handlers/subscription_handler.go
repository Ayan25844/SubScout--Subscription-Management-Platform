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
	"github.com/Ayan25844/subscout/pkg/dto"
	"github.com/Ayan25844/subscout/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SubscriptionHandler struct {
	DB *database.Service
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /subscriptions [post]
// @Summary Create a new subscription
// @Failure 400 {string} string "Invalid input"
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Subscription already exists"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Param request body dto.Subscription_DTO true "New subscription"
// @Success 201 {object} models.Subscription "Subscription created successfully"
// @Description User route to create a unique subscription for the logged in user
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	var errTrial string
	var req dto.Subscription_DTO
	var finalBillingDate *time.Time
	if errMsg := json.NewDecoder(r.Body).Decode(&req); errMsg != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errCatID := validator.ValidateRequiredUUID(req.CategoryID, "Category ID", false); errCatID != "" {
		http.Error(w, errCatID, http.StatusBadRequest)
		return
	}
	if errName := validator.ValidateRequiredString(req.ServiceName, "Service name", false); errName != "" {
		http.Error(w, errName, http.StatusBadRequest)
		return
	}
	if errCycle := validator.ValidateEnum(req.BillingCycle, "Billing cycle", validator.BillingCycles, true); errCycle != "" {
		http.Error(w, errCycle, http.StatusBadRequest)
		return
	}
	if errCost := validator.ValidateGreaterThanZero(req.MonthlyCost, "Monthly cost", false); errCost != "" {
		http.Error(w, errCost, http.StatusBadRequest)
		return
	}
	if finalBillingDate, errTrial = validator.CoalesceBillingDates(req.TrialEndDate, req.NextBillingDate, false); errTrial != "" {
		http.Error(w, errTrial, http.StatusBadRequest)
		return
	}
	var sub models.Subscription
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var err error
	if req.BillingCycle != nil {
		query := `INSERT INTO subscriptions (user_id, category_id, billing_cycle, service_name, monthly_cost, next_billing_date,trial_end_date) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, user_id, status, category_id, billing_cycle, service_name, 
		monthly_cost, created_at, updated_at, next_billing_date, trial_end_date`
		err = h.DB.Pool.QueryRow(ctx, query, claims.UserID, req.CategoryID, req.BillingCycle, req.ServiceName,
			req.MonthlyCost, finalBillingDate, req.TrialEndDate).
			Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.CategoryID, &sub.BillingCycle, &sub.ServiceName, &sub.MonthlyCost,
				&sub.CreatedAt, &sub.UpdatedAt, &sub.NextBillingDate, &sub.TrialEndDate)
	} else {
		query := `INSERT INTO subscriptions (user_id, category_id, service_name, monthly_cost, next_billing_date,trial_end_date) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, status, category_id, billing_cycle, service_name, 
		monthly_cost, created_at, updated_at, next_billing_date, trial_end_date`
		err = h.DB.Pool.QueryRow(ctx, query, claims.UserID, req.CategoryID, req.ServiceName,
			req.MonthlyCost, finalBillingDate, req.TrialEndDate).
			Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.CategoryID, &sub.BillingCycle, &sub.ServiceName, &sub.MonthlyCost,
				&sub.CreatedAt, &sub.UpdatedAt, &sub.NextBillingDate, &sub.TrialEndDate)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Subscription already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /subscriptions [get]
// @Summary Get a list of all subscriptions
// @Failure 500 {string} string "Internal server error"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Failure 404 {string} string "No subscriptions found for this user"
// @Description User route to get a list of all subscriptions for the logged in user
// @Success 200 {array} models.Subscription "List of subscriptions retrieved successfully"
func (h *SubscriptionHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `
		SELECT id, user_id, category_id, service_name, monthly_cost, billing_cycle, next_billing_date, trial_end_date, 
		status, created_at, updated_at
		FROM subscriptions 
		WHERE user_id = $1`
	rows, err := h.DB.Pool.Query(ctx, query, claims.UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		err := rows.Scan(&s.ID, &s.UserID, &s.CategoryID, &s.ServiceName, &s.MonthlyCost, &s.BillingCycle,
			&s.NextBillingDate, &s.TrialEndDate, &s.Status, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(subs) == 0 {
		http.Error(w, "No subscriptions found for this user", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /subscriptions/{categoryID} [get]
// @Param categoryID path string true "Category ID"
// @Failure 500 {string} string "Internal server error"
// @Summary Get a list of subscriptions based on category id
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Failure 404 {string} string "No subscriptions found for this user"
// @Description User route to get a list of all subscriptions for the logged-in user
// @Success 200 {array} models.Subscription "List of Subscriptions retrieved successfully"
func (h *SubscriptionHandler) GetSubscriptionsCategoryID(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	categoryID := chi.URLParam(r, "categoryID")
	if errID := validator.IsValidUUID(&categoryID, "Category ID"); errID != "" {
		http.Error(w, errID, http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `
		SELECT id, user_id, category_id, service_name, monthly_cost, billing_cycle, next_billing_date, trial_end_date, 
		status, created_at, updated_at
		FROM subscriptions 
		WHERE category_id = $1 AND user_id = $2`
	rows, err := h.DB.Pool.Query(ctx, query, categoryID, claims.UserID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		err := rows.Scan(&s.ID, &s.UserID, &s.CategoryID, &s.ServiceName, &s.MonthlyCost, &s.BillingCycle,
			&s.NextBillingDate, &s.TrialEndDate, &s.Status, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(subs) == 0 {
		http.Error(w, "No subscriptions found for this user", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /subscriptions/{id} [put]
// @Summary Update an existing subscription
// @Failure 400 {string} string "Invalid input"
// @Param id path string true "Subscription ID"
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Subscription already exists"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Failure 404 {string} string "No subscription found for this user"
// @Param request body dto.Subscription_DTO true "Updated subscription"
// @Description User route to update a subscription for the logged in user
// @Success 200 {object} models.Subscription "Subscription updated successfully"
func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	subID := chi.URLParam(r, "id")
	if errID := validator.IsValidUUID(&subID, "Subscription ID"); errID != "" {
		http.Error(w, errID, http.StatusBadRequest)
		return
	}
	var errTrial string
	var req dto.Subscription_DTO
	var finalBillingDate *time.Time
	if errMsg := json.NewDecoder(r.Body).Decode(&req); errMsg != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errCatID := validator.ValidateRequiredUUID(req.CategoryID, "Category ID", true); errCatID != "" {
		http.Error(w, errCatID, http.StatusBadRequest)
		return
	}
	if errName := validator.ValidateRequiredString(req.ServiceName, "Service name", true); errName != "" {
		http.Error(w, errName, http.StatusBadRequest)
		return
	}
	if errCycle := validator.ValidateEnum(req.BillingCycle, "Billing cycle", validator.BillingCycles, true); errCycle != "" {
		http.Error(w, errCycle, http.StatusBadRequest)
		return
	}
	if errCost := validator.ValidateGreaterThanZero(req.MonthlyCost, "Monthly cost", true); errCost != "" {
		http.Error(w, errCost, http.StatusBadRequest)
		return
	}
	if finalBillingDate, errTrial = validator.CoalesceBillingDates(req.TrialEndDate, req.NextBillingDate, true); errTrial != "" {
		http.Error(w, errTrial, http.StatusBadRequest)
		return
	}
	var updated models.Subscription
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `
		UPDATE subscriptions 
		SET category_id = COALESCE($1, category_id),
		billing_cycle = COALESCE($2, billing_cycle),
		service_name = COALESCE($3, service_name),
		monthly_cost = COALESCE($4, monthly_cost),
		updated_at = CURRENT_TIMESTAMP,
		next_billing_date = COALESCE($5, next_billing_date),
		trial_end_date = COALESCE($6, trial_end_date)
		WHERE id = $7 AND user_id = $8
		RETURNING id, user_id, status, category_id, billing_cycle, service_name, monthly_cost, created_at, updated_at, 
		next_billing_date, trial_end_date`
	err := h.DB.Pool.QueryRow(ctx, query,
		req.CategoryID, req.BillingCycle, req.ServiceName, req.MonthlyCost,
		finalBillingDate, req.TrialEndDate, subID, claims.UserID,
	).Scan(&updated.ID, &updated.UserID, &updated.Status, &updated.CategoryID, &updated.BillingCycle, &updated.ServiceName,
		&updated.MonthlyCost, &updated.CreatedAt, &updated.UpdatedAt, &updated.NextBillingDate, &updated.TrialEndDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Subscription already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /subscriptions/status/{id} [put]
// @Failure 400 {string} string "Invalid input"
// @Param id path string true "Subscription ID"
// @Summary Update an existing subscription status
// @Failure 500 {string} string "Internal server error"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Failure 404 {string} string "No subscription found for this user"
// @Success 200 {object} models.Subscription "Subscription updated successfully"
// @Description User route to update a subscription status for the logged in user
// @Param request body dto.Update_Subscription_Status true "Updated subscription status"
func (h *SubscriptionHandler) UpdateSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	subID := chi.URLParam(r, "id")
	if errID := validator.IsValidUUID(&subID, "Subscription ID"); errID != "" {
		http.Error(w, errID, http.StatusBadRequest)
		return
	}
	var req dto.Update_Subscription_Status
	if errMsg := json.NewDecoder(r.Body).Decode(&req); errMsg != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errStatus := validator.ValidateEnum(req.Status, "Subscription status", validator.SubscriptionStatus, true); errStatus != "" {
		http.Error(w, errStatus, http.StatusBadRequest)
		return
	}
	var updated models.Subscription
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `
		UPDATE subscriptions 
		SET status = COALESCE($1, status)
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, status, category_id, billing_cycle, service_name, monthly_cost, created_at, updated_at, 
		next_billing_date, trial_end_date`
	err := h.DB.Pool.QueryRow(ctx, query,
		req.Status, subID, claims.UserID,
	).Scan(&updated.ID, &updated.UserID, &updated.Status, &updated.CategoryID, &updated.BillingCycle, &updated.ServiceName,
		&updated.MonthlyCost, &updated.CreatedAt, &updated.UpdatedAt, &updated.NextBillingDate, &updated.TrialEndDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Subscription already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// @Security Bearer
// @Tags User Routes
// @Success 204 "No Content"
// @Router /subscriptions/{id} [delete]
// @Param id path string true "Subscription ID"
// @Summary Delete a subscription based on its id
// @Failure 500 {string} string "Internal server error"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Failure 404 {string} string "No subscription found for this user"
// @Description User route to delete a subscription for the logged-in user
func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	subID := chi.URLParam(r, "id")
	if errID := validator.IsValidUUID(&subID, "Subscription ID"); errID != "" {
		http.Error(w, errID, http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `DELETE FROM subscriptions WHERE id = $1 AND user_id = $2`
	commandTag, err := h.DB.Pool.Exec(ctx, query, subID, claims.UserID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if commandTag.RowsAffected() == 0 {
		http.Error(w, "No subscription found for this user", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags Admin Routes
// @Router /categories [post]
// @Summary Create a new category
// @Failure 400 {string} string "Invalid input"
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Category already exists"
// @Param request body dto.Category_DTO true "Category Name"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Description Admin route to create a unique subscription category
// @Failure 403 {string} string "Forbidden - Insufficient Permissions"
// @Success 201 {object} models.Category "Category created successfully"
func (h *SubscriptionHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	var req dto.Category_DTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errMsg := validator.ValidateRequiredString(req.Name, "Category name", false); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	var category models.Category
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `INSERT INTO categories (name, created_by, updated_by) VALUES ($1, $2, $3) RETURNING id, name, created_by, 
	updated_by, created_at, updated_at`
	err := h.DB.Pool.QueryRow(ctx, query, req.Name, claims.UserID, claims.UserID).Scan(&category.ID, &category.Name,
		&category.CreatedBy, &category.UpdatedBy, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Category already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

// @Produce json
// @Security Bearer
// @Tags User Routes
// @Router /categories [get]
// @Summary Get the list of all categories
// @Failure 404 {string} string "No categories found"
// @Failure 500 {string} string "Internal server error"
// @Description User route to get a list of all categories
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Success 200 {array} models.Category "List of categories retrieved successfully"
func (h *SubscriptionHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `SELECT id, name, created_by, updated_by, created_at, updated_at FROM categories`
	rows, err := h.DB.Pool.Query(ctx, query)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedBy, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(categories) == 0 {
		http.Error(w, "No categories found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// @Accept json
// @Produce json
// @Security Bearer
// @Tags Admin Routes
// @Router /categories/{id} [put]
// @Summary Update the name of a category
// @Param id path string true "Category ID"
// @Failure 400 {string} string "Invalid input"
// @Description Admin route to rename a category
// @Failure 404 {string} string "Category not found"
// @Failure 500 {string} string "Internal server error"
// @Failure 409 {string} string "Category already exists"
// @Failure 401 {string} string "Unauthorized: Missing token"
// @Param request body dto.Category_DTO true "Renamed category"
// @Failure 403 {string} string "Forbidden: Insufficient Permissions"
// @Success 200 {object} models.Category "Category updated successfully"
func (h *SubscriptionHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserContextKey).(*middleware.CustomClaims)
	categoryID := chi.URLParam(r, "id")
	if errID := validator.IsValidUUID(&categoryID, "Category ID"); errID != "" {
		http.Error(w, errID, http.StatusBadRequest)
		return
	}
	var req dto.Category_DTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if errMsg := validator.ValidateRequiredString(req.Name, "Category Name", true); errMsg != "" {
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	var updatedCategory models.Category
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	query := `UPDATE categories SET name = COALESCE($1, name), updated_by = $2 , updated_at = CURRENT_TIMESTAMP WHERE id = $3 RETURNING id, 
	name, created_by, updated_by, created_at, updated_at`
	err := h.DB.Pool.QueryRow(ctx, query, req.Name, claims.UserID, categoryID).Scan(&updatedCategory.ID,
		&updatedCategory.Name, &updatedCategory.CreatedBy, &updatedCategory.UpdatedBy, &updatedCategory.CreatedAt,
		&updatedCategory.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Category not found", http.StatusNotFound)
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Category already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCategory)
}
