package dto

type SuspendAccountResponse struct {
	UserID      string `json:"userId"`
	Status      string `json:"status"`
	SuspendedAt string `json:"suspendedAt"`
}
