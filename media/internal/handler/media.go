package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tafu-media/media-service/internal/domain"
	"github.com/tafu-media/media-service/internal/usecase"
)

type MediaHandler struct {
	useCase *usecase.MediaUseCase
}

func NewMediaHandler(useCase *usecase.MediaUseCase) *MediaHandler {
	return &MediaHandler{useCase: useCase}
}

type GetUploadURLRequest struct {
	UserID   string `json:"user_id"`
	FileType string `json:"file_type"`
	Purpose  string `json:"purpose"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func (h *MediaHandler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var req GetUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "missing_user_id", "user_id is required")
		return
	}
	if req.FileType == "" {
		writeError(w, http.StatusBadRequest, "missing_file_type", "file_type is required")
		return
	}

	result, err := h.useCase.GetUploadURL(r.Context(), domain.UploadRequest{
		UserID:   req.UserID,
		FileType: req.FileType,
		Purpose:  req.Purpose,
	})
	if err != nil {
		if err == domain.ErrInvalidFileType {
			writeError(w, http.StatusBadRequest, "invalid_file_type", "File type not allowed")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MediaHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var req domain.ConfirmUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.FileKey == "" {
		writeError(w, http.StatusBadRequest, "missing_file_key", "file_key is required")
		return
	}

	err := h.useCase.ConfirmUpload(r.Context(), req)
	if err != nil {
		if err == domain.ErrFileNotFound {
			writeError(w, http.StatusNotFound, "file_not_found", "File not found in storage")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

func (h *MediaHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, errorCode, message string) {
	writeJSON(w, status, ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
