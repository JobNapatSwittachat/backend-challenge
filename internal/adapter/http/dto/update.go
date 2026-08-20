package dto

// UpdateUserRequest is the body of PATCH /users/{id}. A nil field means the
// client omitted it and the value is left unchanged.
type UpdateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}
