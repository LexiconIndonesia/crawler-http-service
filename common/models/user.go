package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id" validate:"required,uuid"`
	Email     string    `json:"email" validate:"required,email"`
	FirstName string    `json:"first_name" validate:"required"`
	LastName  string    `json:"last_name" validate:"required"`
	Password  string    `json:"-" validate:"required,min=8"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserValidator is a validator for user models
type UserValidator struct {
	validator *validator.Validate
}

// NewUserValidator creates a new user validator
func NewUserValidator() *UserValidator {
	return &UserValidator{
		validator: validator.New(),
	}
}

// Validate validates a user model
func (v *UserValidator) Validate(user User) error {
	return v.validator.Struct(user)
}

// ValidatePartial validates a partial user model (for updates)
func (v *UserValidator) ValidatePartial(user User) error {
	return v.validator.StructPartial(user, "Email", "FirstName", "LastName")
}
