package validator

import (
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

var BillingCycles = map[string]bool{
	"daily": true, "weekly": true, "monthly": true, "quarterly": true, "yearly": true,
}

var SubscriptionStatus = map[string]bool{
	"active": true, "paused": true, "cancelled": true, "pastdue": true,
}

func IsFutureDate(date time.Time) bool {
	return date.After(time.Now().UTC())
}

func ValidatePasswordStrength(password string) bool {
	var (
		hasMinLen  = len(password) >= 8
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasMinLen && hasUpper && hasLower && hasNumber && hasSpecial
}

func IsEmpty(input *string, fieldName string, isUpdate bool) (string, bool) {
	if isUpdate && input == nil {
		return "", true
	}
	if input == nil || strings.TrimSpace(*input) == "" {
		return fieldName + " cannot be empty", true
	}
	return "", false
}

func IsValidUUID(input *string, fieldName string) string {
	if _, err := uuid.Parse(*input); err != nil {
		return "Invalid " + fieldName + " format"
	}
	return ""
}

func IsValidEmail(email *string, fieldName string, isUpdate bool) string {
	if errMsg, stop := IsEmpty(email, fieldName, isUpdate); stop {
		return errMsg
	}
	if _, errMsg := mail.ParseAddress(*email); errMsg != nil {
		return "Invalid email id"
	}
	return ""
}

func ValidateGreaterThanZero(input *float64, fieldName string, isUpdate bool) string {
	if input == nil {
		if isUpdate {
			return ""
		}
		return fieldName + " cannot be empty"
	}
	if *input <= 0 {
		return fieldName + " must be greater than zero"
	}
	return ""
}

func ValidateRequiredUUID(input *uuid.UUID, fieldName string, isUpdate bool) string {
	if isUpdate && input == nil {
		return ""
	}
	if input == nil {
		return fieldName + " cannot be empty"
	}
	return ""
}

func ValidateRequiredString(input *string, fieldName string, isUpdate bool) string {
	if errMsg, stop := IsEmpty(input, fieldName, isUpdate); stop {
		return errMsg
	}
	return ""
}

func CoalesceBillingDates(trialEnd *time.Time, nextBilling *time.Time, isUpdate bool) (*time.Time, string) {
	if nextBilling == nil {
		if trialEnd == nil {
			if isUpdate {
				return nil, ""
			}
			return nil, "Next billing date cannot not be empty"
		}
		if !IsFutureDate(*trialEnd) {
			return nil, "Trial end date must be a future date"
		}
		return trialEnd, ""
	}
	if !IsFutureDate(*nextBilling) {
		return nil, "Next billing date must be a future date"
	}
	if trialEnd != nil {
		if !IsFutureDate(*trialEnd) {
			return nil, "Trial end date must be a future date"
		}
		if !nextBilling.After(*trialEnd) {
			return nil, "Trial must end before the next billing date"
		}
	}
	return nextBilling, ""
}

func ValidateEnum(input *string, fieldName string, validValues map[string]bool, isUpdate bool) string {
	if errMsg, stop := IsEmpty(input, fieldName, isUpdate); stop {
		return errMsg
	}
	if !validValues[*input] {
		return "Invalid " + fieldName
	}
	return ""
}

func ValidatePassword(oldPassword *string, newPassword *string, isUpdate bool) string {
	if isUpdate {
		if errMsg, stop := IsEmpty(oldPassword, "Password fields", false); stop {
			return errMsg
		}
		if errMsg, stop := IsEmpty(newPassword, "Password fields", false); stop {
			return errMsg
		}
		if *oldPassword == *newPassword {
			return "New password cannot be the same as the old password"
		}
	} else {
		if errMsg, stop := IsEmpty(newPassword, "Password", false); stop {
			return errMsg
		}
	}
	if !ValidatePasswordStrength(*newPassword) {
		return "Invalid password"
	}
	return ""
}
