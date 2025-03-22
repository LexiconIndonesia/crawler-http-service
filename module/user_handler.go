package module

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/adryanev/go-http-service-template/common/models"
	"github.com/adryanev/go-http-service-template/common/utils"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// userHandler handles user-related requests
func (m *Module) userHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		m.createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// UserResponse is the response for user endpoints
type UserResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string    `json:"email" example:"user@example.com"`
	FirstName string    `json:"first_name" example:"John"`
	LastName  string    `json:"last_name" example:"Doe"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// UserCreationRequest represents the request to create a user
type UserCreationRequest struct {
	Email     string `json:"email" example:"user@example.com" validate:"required,email"`
	FirstName string `json:"first_name" example:"John" validate:"required"`
	LastName  string `json:"last_name" example:"Doe" validate:"required"`
	Password  string `json:"password" example:"Password123!" validate:"required,min=8"`
}

// @Summary Create a new user
// @Description Create a new user with the provided information
// @Tags users
// @Accept json
// @Produce json
// @Param request body UserCreationRequest true "User creation request"
// @Success 201 {object} utils.Response{data=UserResponse} "User created successfully"
// @Failure 400 {object} utils.Response{error=string} "Invalid request body"
// @Failure 500 {object} utils.Response{error=string} "Internal server error"
// @Router /users [post]
func (m *Module) createUser(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var userReq UserCreationRequest
	if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
		log.Error().Err(err).Msg("Failed to decode request body")
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Convert to user model
	user := models.User{
		ID:        uuid.New().String(),
		Email:     userReq.Email,
		FirstName: userReq.FirstName,
		LastName:  userReq.LastName,
		Password:  userReq.Password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Validate user
	validator := models.NewUserValidator()
	if err := validator.Validate(user); err != nil {
		log.Error().Err(err).Msg("Invalid user data")
		utils.WriteError(w, http.StatusBadRequest, "Invalid user data: "+err.Error())
		return
	}

	// TODO: Save user to database
	// In a real implementation, you would save the user to the database here
	// using the dependency-injected DB

	// Create response object (omitting password)
	userResponse := UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	// Return success
	utils.WriteJSON(w, http.StatusCreated, userResponse)
}
