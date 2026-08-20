package dto

// RegisterRequest is the body of POST /auth/register and POST /users.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
