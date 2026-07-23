package dto

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type AuthUserResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}
type RegisterResponse struct {
	User      AuthUserResponse `json:"user"`
	CreatedAt string           `json:"createdAt"`
}
type LoginResponse struct {
	User        AuthUserResponse `json:"user"`
	AccessToken string           `json:"accessToken"`
	TokenType   string           `json:"tokenType"`
	ExpiresAt   string           `json:"expiresAt"`
}
type RefreshResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresAt   string `json:"expiresAt"`
}
type LogoutResponse struct {
	Message string `json:"message"`
}
type MeResponse struct {
	User AuthUserResponse `json:"user"`
}
