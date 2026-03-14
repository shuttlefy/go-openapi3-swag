package simple

// UserResponse represents a user in the system.
type UserResponse struct {
	ID    int      `json:"id"`
	Name  string   `json:"name" binding:"required"`
	Email *string  `json:"email,omitempty"`
	Tags  []string `json:"tags"`
}

// ErrorResponse represents an error.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetUser godoc
// @Summary     Get user by ID
// @Description Get a user by their ID
// @Tags        users
// @Param       id path int true "User ID"
// @Success     200 {object} UserResponse
// @Failure     404 {object} ErrorResponse
// @Router      /users/{id} [get]
func GetUser(id int) (*UserResponse, error) {
	return nil, nil
}

type UserController struct{}

// CreateUser godoc
// @Summary     Create a new user
// @Description Create a new user with the given information
// @Tags        users
// @Param       body body UserResponse true "User to create"
// @Success     201 {object} UserResponse
// @Router      /users [post]
func (c *UserController) CreateUser(user UserResponse) (*UserResponse, error) {
	return nil, nil
}
