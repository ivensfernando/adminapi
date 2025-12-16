package model

const userTimeLayout = "2006-01-02 15:04:05"

type UpdateUserPayload struct {
	Email     *string `json:"email,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type ChangePasswordPayload struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UserResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Bio       string `json:"bio,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	LastLogin string `json:"last_login,omitempty"`
	LastSeen  string `json:"last_seen,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	resp := UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Bio:       u.Bio,
		AvatarURL: u.AvatarURL,
	}

	if !u.LastLogin.IsZero() {
		resp.LastLogin = u.LastLogin.Format(userTimeLayout)
	}

	if !u.LastSeen.IsZero() {
		resp.LastSeen = u.LastSeen.Format(userTimeLayout)
	}

	if !u.CreatedAt.IsZero() {
		resp.CreatedAt = u.CreatedAt.Format(userTimeLayout)
	}

	if !u.UpdatedAt.IsZero() {
		resp.UpdatedAt = u.UpdatedAt.Format(userTimeLayout)
	}

	return resp
}

func NewUserResponse(user *User) UserResponse {
	if user == nil {
		return UserResponse{}
	}
	return user.ToResponse()
}
