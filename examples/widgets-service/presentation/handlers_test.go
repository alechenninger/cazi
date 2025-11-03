package presentation_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"widgets-service/application"
	"widgets-service/infrastructure"
	"widgets-service/presentation"
)

func setupHandler() *presentation.WidgetHandler {
	repo := infrastructure.NewInMemoryWidgetRepository()
	authz := infrastructure.NewLocalAuthz()
	service := application.NewWidgetService(repo, authz)
	return presentation.NewWidgetHandler(service)
}

func TestCreateAndGetOwnWidget(t *testing.T) {
	handler := setupHandler()

	// Create a widget as user-alice
	createReq := map[string]string{
		"id":          "widget-1",
		"name":        "Test Widget",
		"description": "A test widget",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/widgets", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user-alice")
	rec := httptest.NewRecorder()

	handler.CreateWidget(rec, req)

	// Assert create succeeded
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp presentation.WidgetHTTPResponse
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if createResp.ID != "widget-1" {
		t.Errorf("expected ID 'widget-1', got '%s'", createResp.ID)
	}
	if createResp.OwnerID != "user-alice" {
		t.Errorf("expected owner 'user-alice', got '%s'", createResp.OwnerID)
	}

	// Now retrieve the widget as the same user
	getReq := httptest.NewRequest(http.MethodGet, "/widgets/widget-1", nil)
	getReq.Header.Set("X-User-ID", "user-alice")
	getRec := httptest.NewRecorder()

	handler.GetWidget(getRec, getReq)

	// Assert retrieval succeeded
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}

	var getResp presentation.WidgetHTTPResponse
	if err := json.NewDecoder(getRec.Body).Decode(&getResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if getResp.ID != "widget-1" {
		t.Errorf("expected ID 'widget-1', got '%s'", getResp.ID)
	}
	if getResp.Name != "Test Widget" {
		t.Errorf("expected name 'Test Widget', got '%s'", getResp.Name)
	}
	if getResp.OwnerID != "user-alice" {
		t.Errorf("expected owner 'user-alice', got '%s'", getResp.OwnerID)
	}
}

func TestCreateWidget_MissingAuth(t *testing.T) {
	handler := setupHandler()

	createReq := map[string]string{
		"id":          "widget-1",
		"name":        "Test Widget",
		"description": "A test widget",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/widgets", bytes.NewReader(body))
	// No X-User-ID header set
	rec := httptest.NewRecorder()

	handler.CreateWidget(rec, req)

	// Assert unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetWidget_OtherUsersWidget(t *testing.T) {
	handler := setupHandler()

	// User Alice creates a widget
	createReq := map[string]string{
		"id":          "widget-alice",
		"name":        "Alice's Widget",
		"description": "Private to Alice",
	}
	body, _ := json.Marshal(createReq)

	req := httptest.NewRequest(http.MethodPost, "/widgets", bytes.NewReader(body))
	req.Header.Set("X-User-ID", "user-alice")
	rec := httptest.NewRecorder()

	handler.CreateWidget(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d: %s", rec.Code, rec.Body.String())
	}

	// User Bob tries to retrieve Alice's widget
	getReq := httptest.NewRequest(http.MethodGet, "/widgets/widget-alice", nil)
	getReq.Header.Set("X-User-ID", "user-bob")
	getRec := httptest.NewRecorder()

	handler.GetWidget(getRec, getReq)

	// Assert not found - demonstrates information hiding
	// Bob can't distinguish between "doesn't exist" and "not authorized"
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestMultipleUsers_Authorization(t *testing.T) {
	handler := setupHandler()

	// User Alice creates widget-1
	createReq1 := map[string]string{
		"id":          "widget-1",
		"name":        "Widget 1",
		"description": "Alice's widget",
	}
	body1, _ := json.Marshal(createReq1)
	req1 := httptest.NewRequest(http.MethodPost, "/widgets", bytes.NewReader(body1))
	req1.Header.Set("X-User-ID", "user-alice")
	rec1 := httptest.NewRecorder()
	handler.CreateWidget(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("alice create failed: %d", rec1.Code)
	}

	// User Bob creates widget-2
	createReq2 := map[string]string{
		"id":          "widget-2",
		"name":        "Widget 2",
		"description": "Bob's widget",
	}
	body2, _ := json.Marshal(createReq2)
	req2 := httptest.NewRequest(http.MethodPost, "/widgets", bytes.NewReader(body2))
	req2.Header.Set("X-User-ID", "user-bob")
	rec2 := httptest.NewRecorder()
	handler.CreateWidget(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("bob create failed: %d", rec2.Code)
	}

	// Alice can get widget-1 (her own)
	getReq := httptest.NewRequest(http.MethodGet, "/widgets/widget-1", nil)
	getReq.Header.Set("X-User-ID", "user-alice")
	getRec := httptest.NewRecorder()
	handler.GetWidget(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("alice should access widget-1: got %d", getRec.Code)
	}

	// Alice cannot get widget-2 (Bob's)
	getReq = httptest.NewRequest(http.MethodGet, "/widgets/widget-2", nil)
	getReq.Header.Set("X-User-ID", "user-alice")
	getRec = httptest.NewRecorder()
	handler.GetWidget(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Errorf("alice should not access widget-2: got %d", getRec.Code)
	}

	// Bob can get widget-2 (his own)
	getReq = httptest.NewRequest(http.MethodGet, "/widgets/widget-2", nil)
	getReq.Header.Set("X-User-ID", "user-bob")
	getRec = httptest.NewRecorder()
	handler.GetWidget(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("bob should access widget-2: got %d", getRec.Code)
	}

	// Bob cannot get widget-1 (Alice's)
	getReq = httptest.NewRequest(http.MethodGet, "/widgets/widget-1", nil)
	getReq.Header.Set("X-User-ID", "user-bob")
	getRec = httptest.NewRecorder()
	handler.GetWidget(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Errorf("bob should not access widget-1: got %d", getRec.Code)
	}
}
