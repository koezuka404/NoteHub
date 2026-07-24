package dto

type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

type CreateWorkspaceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	HostID    string `json:"hostId"`
	CreatedAt string `json:"createdAt"`
}

type WorkspaceListItemResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	HostID            string `json:"hostId"`
	HostName          string `json:"hostName"`
	Role              string `json:"role"`
	IsAvailable       bool   `json:"isAvailable"`
	UnavailableReason string `json:"unavailableReason"`
	UpdatedAt         string `json:"updatedAt"`
}

type WorkspaceHostResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type GetWorkspaceResponse struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Host      WorkspaceHostResponse `json:"host"`
	Role      string                `json:"role"`
	CreatedAt string                `json:"createdAt"`
	UpdatedAt string                `json:"updatedAt"`
}

type UpdateWorkspaceRequest struct {
	Name string `json:"name"`
}

type UpdateWorkspaceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
}

type DeleteWorkspaceRequest struct {
	Reason string `json:"reason"`
}

type DeleteWorkspaceResponse struct {
	WorkspaceID string `json:"workspaceId"`
	DeletedAt   string `json:"deletedAt"`
}
