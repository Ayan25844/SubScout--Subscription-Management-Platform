package dto

import (
	"time"

	"github.com/google/uuid"
)

type User_DTO struct {
	Currency *string `json:"currency" swaggertype:"string" example:"INR"`
	Password *string `json:"password" swaggertype:"string" format:"password" validate:"required"`
	Email    *string `json:"email" swaggertype:"string" format:"email" example:"xyz@gmail.com" validate:"required"`
}

type Login struct {
	Password *string `json:"password" swaggertype:"string" format:"password" validate:"required"`
	Email    *string `json:"email" swaggertype:"string" format:"email" example:"xyz@gmail.com" validate:"required"`
}

type Category_DTO struct {
	Name *string `json:"name" swaggertype:"string" example:"Cloud" validate:"required"`
}

type Subscription_DTO struct {
	CategoryID      *uuid.UUID `json:"category_id" swaggertype:"string" format:"uuid" validate:"required"`
	BillingCycle    *string    `json:"billing_cycle" swaggertype:"string" example:"monthly"`
	ServiceName     *string    `json:"service_name" swaggertype:"string" example:"Catalog service" validate:"required"`
	MonthlyCost     *float64   `json:"monthly_cost" swaggertype:"number" format:"float" example:"15.99" validate:"required"`
	NextBillingDate *time.Time `json:"next_billing_date" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z" validate:"required"`
	TrialEndDate    *time.Time `json:"trial_end_date,omitempty" swaggertype:"string" format:"date-time" example:"2026-03-09T12:00:00Z"`
}

type Update_Subscription_Status struct {
	Status *string `json:"status" swaggertype:"string" example:"active"`
}

type Update_User_Profile struct {
	Currency *string `json:"currency" swaggertype:"string" example:"INR"`
	Email    *string `json:"email" swaggertype:"string" format:"email" example:"xyz@gmail.com"`
}

type Update_User_Password struct {
	OldPassword *string `json:"old_password" swaggertype:"string" format:"password"`
	NewPassword *string `json:"new_password" swaggertype:"string" format:"password"`
}
