package dto

type SearchUserResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type MemberResponse struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}

type AddMemberRequest struct {
	UserID string `json:"userId"`
}

type AddMemberResponse struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}
