package dto

type SuspendAccountResponse struct {
	UserID      string `json:"userId"`
	Status      string `json:"status"`
	SuspendedAt string `json:"suspendedAt"`
}

type ReactivateAccountResponse struct {
	UserID string `json:"userId"`
	Status string `json:"status"`
}

type DeleteAccountResponse struct {
	UserID    string `json:"userId"`
	Status    string `json:"status"`
	DeletedAt string `json:"deletedAt"`
}
