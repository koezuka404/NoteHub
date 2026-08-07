package dto

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

func NewErrorResponse(code, message, requestID string) ErrorResponse {
	body := ErrorBody{
		Code:    code,
		Message: message,
	}
	if requestID != "" {
		body.RequestID = requestID
	}
	return ErrorResponse{Error: body}
}
