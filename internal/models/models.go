package models

import (
	"time"

	"github.com/google/uuid"
)

// @Description Detailed information about a user's profile
type User struct {
	ID        uuid.UUID `json:"id" swaggertype:"string" format:"uuid"`
	Role      string    `json:"role" swaggertype:"string" example:"admin"`
	Currency  string    `json:"currency" swaggertype:"string" example:"INR"`
	Email     string    `json:"email" swaggertype:"string" format:"email" example:"xyz@gmail.com"`
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
}

// @Description Metadata used to organize and filter subscriptions
type Category struct {
	ID        uuid.UUID `json:"id" swaggertype:"string" format:"uuid"`
	Name      string    `json:"name" swaggertype:"string" example:"Cloud"`
	CreatedBy uuid.UUID `json:"created_by" swaggertype:"string" format:"uuid"`
	UpdatedBy uuid.UUID `json:"updated_by" swaggertype:"string" format:"uuid"`
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
}

// @Description Detailed information about a user's subscription
type Subscription struct {
	ID              uuid.UUID  `json:"id" swaggertype:"string" format:"uuid"`
	UserID          uuid.UUID  `json:"user_id" swaggertype:"string" format:"uuid"`
	Status          string     `json:"status" swaggertype:"string" example:"active"`
	CategoryID      uuid.UUID  `json:"category_id" swaggertype:"string" format:"uuid"`
	BillingCycle    string     `json:"billing_cycle" swaggertype:"string" example:"monthly"`
	ServiceName     string     `json:"service_name" swaggertype:"string" example:"Catalog service"`
	MonthlyCost     float64    `json:"monthly_cost" swaggertype:"number" format:"float" example:"15.99"`
	CreatedAt       time.Time  `json:"created_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
	UpdatedAt       time.Time  `json:"updated_at" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
	NextBillingDate time.Time  `json:"next_billing_date" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
	TrialEndDate    *time.Time `json:"trial_end_date,omitempty" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
}
