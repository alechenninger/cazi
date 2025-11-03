package presentation

import (
	"encoding/json"
	"net/http"

	"widgets-service/application"

	"github.com/alechenninger/cazi/pkg/cazi"
)

// WidgetHandler provides HTTP handlers for widget operations.
type WidgetHandler struct {
	service *application.WidgetService
}

// NewWidgetHandler creates a new widget HTTP handler.
func NewWidgetHandler(service *application.WidgetService) *WidgetHandler {
	return &WidgetHandler{service: service}
}

// CreateWidgetHTTPRequest is the HTTP request body for creating a widget.
type CreateWidgetHTTPRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// WidgetHTTPResponse is the HTTP response for widget operations.
type WidgetHTTPResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"owner_id"`
}

// ErrorResponse represents an HTTP error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateWidget handles POST /widgets requests.
func (h *WidgetHandler) CreateWidget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from header and construct Subject (simplified authentication)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	subject := cazi.Subject{
		Assertion: cazi.ResourceReference{Type: "user", ID: userID},
	}

	// Decode request
	var httpReq CreateWidgetHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call application service
	appReq := application.CreateWidgetRequest{
		Subject:     subject,
		WidgetID:    httpReq.ID,
		Name:        httpReq.Name,
		Description: httpReq.Description,
	}

	widget, err := h.service.CreateWidget(r.Context(), appReq)
	if err != nil {
		h.writeError(w, http.StatusForbidden, err.Error())
		return
	}

	// Write response
	h.writeJSON(w, http.StatusCreated, WidgetHTTPResponse{
		ID:          widget.ID,
		Name:        widget.Name,
		Description: widget.Description,
		OwnerID:     widget.OwnerID,
	})
}

// GetWidget handles GET /widgets/{id} requests.
func (h *WidgetHandler) GetWidget(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from header and construct Subject (simplified authentication)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		h.writeError(w, http.StatusUnauthorized, "missing X-User-ID header")
		return
	}

	subject := cazi.Subject{
		Assertion: cazi.ResourceReference{Type: "user", ID: userID},
	}

	// Extract widget ID from path
	widgetID := r.URL.Path[len("/widgets/"):]
	if widgetID == "" {
		h.writeError(w, http.StatusBadRequest, "missing widget ID")
		return
	}

	// Call application service
	appReq := application.GetWidgetRequest{
		Subject:  subject,
		WidgetID: widgetID,
	}

	widget, err := h.service.GetWidget(r.Context(), appReq)
	if err != nil {
		h.writeError(w, http.StatusForbidden, err.Error())
		return
	}

	// Write response
	h.writeJSON(w, http.StatusOK, WidgetHTTPResponse{
		ID:          widget.ID,
		Name:        widget.Name,
		Description: widget.Description,
		OwnerID:     widget.OwnerID,
	})
}

func (h *WidgetHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WidgetHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{Error: message})
}
