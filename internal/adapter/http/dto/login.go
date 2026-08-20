package dto

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse carries the issued JWT and the authenticated user.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
