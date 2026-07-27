package dto

type VersionListItemResponse struct {
	ID        string `json:"id"`
	Type      string `json:"versionType"`
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
}

type GetVersionResponse struct {
	ID         string `json:"id"`
	DocumentID string `json:"documentId"`
	Content    string `json:"content"`
	Type       string `json:"versionType"`
	CreatedBy  string `json:"createdBy"`
	CreatedAt  string `json:"createdAt"`
}

type RestoreVersionResponse struct {
	DocumentID string `json:"documentId"`
	VersionID  string `json:"versionId"`
	RestoredAt string `json:"restoredAt"`
}
