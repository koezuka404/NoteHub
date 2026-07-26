package dto

type CreateDocumentRequest struct {
	Title string `json:"title"`
}

type CreateDocumentResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	CreatedBy   string `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
}

type DocumentListItemResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedBy string `json:"updatedBy"`
	UpdatedAt string `json:"updatedAt"`
}

type GetDocumentResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	UpdatedBy   string `json:"updatedBy"`
	UpdatedAt   string `json:"updatedAt"`
}

type UpdateDocumentRequest struct {
	Title string `json:"title"`
}

type UpdateDocumentResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
}

type DeleteDocumentResponse struct {
	DocumentID string `json:"documentId"`
	DeletedAt  string `json:"deletedAt"`
}
