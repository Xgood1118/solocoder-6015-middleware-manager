package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hhftechnology/middleware-manager/database"
	"github.com/hhftechnology/middleware-manager/internal/testutil"
)

// TestNewMiddlewareHandler tests middleware handler creation
func TestNewMiddlewareHandler(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	if handler == nil {
		t.Fatal("NewMiddlewareHandler() returned nil")
	}
	if handler.DB == nil {
		t.Error("handler.DB is nil")
	}
}

// TestMiddlewareHandler_GetMiddlewares tests fetching all middlewares
func TestMiddlewareHandler_GetMiddlewares(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	// Insert test middlewares
	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('mw-1', 'rate-limiter', 'rateLimit', '{"average":100,"burst":50}')
	`)
	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('mw-2', 'custom-headers', 'headers', '{"customRequestHeaders":{"X-Custom":"value"}}')
	`)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares", nil)
	handler.GetMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var middlewares []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &middlewares); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(middlewares))
	}
}

// TestMiddlewareHandler_GetMiddlewares_Pagination tests paginated results
func TestMiddlewareHandler_GetMiddlewares_Pagination(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	// Insert 5 test middlewares
	for i := 1; i <= 5; i++ {
		testutil.MustExec(t, db, `
			INSERT INTO middlewares (id, name, type, config)
			VALUES (?, ?, 'headers', '{}')
		`, "mw-"+string(rune('0'+i)), "middleware-"+string(rune('0'+i)))
	}

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares?page=1&page_size=2", nil)
	handler.GetMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	// Check pagination metadata
	if response["total"] == nil {
		t.Error("expected total in paginated response")
	}
	if response["page"] == nil {
		t.Error("expected page in paginated response")
	}
	if response["data"] == nil {
		t.Error("expected data array in paginated response")
	}
}

// TestMiddlewareHandler_GetMiddleware tests fetching a single middleware
func TestMiddlewareHandler_GetMiddleware(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('test-mw-id', 'test-middleware', 'headers', '{"customRequestHeaders":{"X-Test":"1"}}')
	`)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/test-mw-id", nil)
	c.Params = gin.Params{{Key: "id", Value: "test-mw-id"}}
	handler.GetMiddleware(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var middleware map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &middleware)

	if middleware["name"] != "test-middleware" {
		t.Errorf("expected name test-middleware, got %v", middleware["name"])
	}
	if middleware["type"] != "headers" {
		t.Errorf("expected type headers, got %v", middleware["type"])
	}
}

// TestMiddlewareHandler_GetMiddleware_NotFound tests fetching non-existent middleware
func TestMiddlewareHandler_GetMiddleware_NotFound(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/non-existent", nil)
	c.Params = gin.Params{{Key: "id", Value: "non-existent"}}
	handler.GetMiddleware(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// TestMiddlewareHandler_GetMiddleware_EmptyID tests missing middleware ID
func TestMiddlewareHandler_GetMiddleware_EmptyID(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	handler.GetMiddleware(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestMiddlewareHandler_CreateMiddleware tests creating a new middleware
func TestMiddlewareHandler_CreateMiddleware(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	body := bytes.NewBufferString(`{
		"name": "new-middleware",
		"type": "headers",
		"config": {
			"customRequestHeaders": {"X-New": "value"}
		}
	}`)

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares", body)
	handler.CreateMiddleware(c)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created)

	if created["id"] == nil || created["id"] == "" {
		t.Error("expected generated ID")
	}
	if created["name"] != "new-middleware" {
		t.Errorf("expected name new-middleware, got %v", created["name"])
	}
}

// TestMiddlewareHandler_CreateMiddleware_InvalidType tests invalid middleware type
func TestMiddlewareHandler_CreateMiddleware_InvalidType(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	body := bytes.NewBufferString(`{
		"name": "invalid-middleware",
		"type": "invalidType",
		"config": {}
	}`)

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares", body)
	handler.CreateMiddleware(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestMiddlewareHandler_CreateMiddleware_ValidationError tests missing required fields
func TestMiddlewareHandler_CreateMiddleware_ValidationError(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	// Missing name field
	body := bytes.NewBufferString(`{
		"type": "headers",
		"config": {}
	}`)

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares", body)
	handler.CreateMiddleware(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestMiddlewareHandler_UpdateMiddleware tests updating a middleware
func TestMiddlewareHandler_UpdateMiddleware(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	// Create middleware first
	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('update-test', 'old-name', 'headers', '{}')
	`)

	body := bytes.NewBufferString(`{
		"name": "updated-name",
		"type": "headers",
		"config": {"customRequestHeaders": {"X-Updated": "true"}}
	}`)

	c, rec := testutil.NewContext(t, http.MethodPut, "/api/middlewares/update-test", body)
	c.Params = gin.Params{{Key: "id", Value: "update-test"}}
	handler.UpdateMiddleware(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &updated)

	if updated["name"] != "updated-name" {
		t.Errorf("expected updated name, got %v", updated["name"])
	}
}

// TestMiddlewareHandler_DeleteMiddleware tests deleting a middleware
func TestMiddlewareHandler_DeleteMiddleware(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('delete-test', 'delete-me', 'headers', '{}')
	`)

	c, rec := testutil.NewContext(t, http.MethodDelete, "/api/middlewares/delete-test", nil)
	c.Params = gin.Params{{Key: "id", Value: "delete-test"}}
	handler.DeleteMiddleware(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify middleware is deleted
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM middlewares WHERE id = 'delete-test'").Scan(&count)
	if count != 0 {
		t.Error("middleware was not deleted")
	}

	// Verify deleted_templates entry was created
	var templateCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM deleted_templates WHERE id = 'delete-test' AND type = 'middleware'").Scan(&templateCount)
	if templateCount != 1 {
		t.Error("deleted_templates entry was not created")
	}
}

// TestMiddlewareHandler_DeleteMiddleware_NotFound tests deleting non-existent middleware
func TestMiddlewareHandler_DeleteMiddleware_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "middleware-delete-notfound")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init temp db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tmpDir)
	})

	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodDelete, "/api/middlewares/non-existent", nil)
	c.Params = gin.Params{{Key: "id", Value: "non-existent"}}
	handler.DeleteMiddleware(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// TestMiddlewareHandler_GetMiddlewares_Empty tests empty database
func TestMiddlewareHandler_GetMiddlewares_Empty(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares", nil)
	handler.GetMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var middlewares []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &middlewares)

	// Should return empty array, not null
	if middlewares == nil {
		t.Error("expected empty array, got nil")
	}
}

// TestMiddlewareHandler_GetMiddlewares_ConfigParsing tests config JSON parsing
func TestMiddlewareHandler_GetMiddlewares_ConfigParsing(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config)
		VALUES ('mw-config', 'config-test', 'rateLimit', '{"average":100,"burst":50,"period":"1s"}')
	`)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares", nil)
	handler.GetMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var middlewares []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &middlewares)

	if len(middlewares) != 1 {
		t.Fatalf("expected 1 middleware, got %d", len(middlewares))
	}

	config, ok := middlewares[0]["config"].(map[string]interface{})
	if !ok {
		t.Fatal("config should be a map")
	}

	if config["average"] == nil {
		t.Error("config should contain 'average' field")
	}
	if config["burst"] == nil {
		t.Error("config should contain 'burst' field")
	}
}

// TestMiddlewareHandler_ValidMiddlewareTypes tests all valid middleware types
func TestMiddlewareHandler_ValidMiddlewareTypes(t *testing.T) {
	validTypes := []string{
		"basicAuth",
		"digestAuth",
		"forwardAuth",
		"ipAllowList",
		"rateLimit",
		"headers",
		"stripPrefix",
		"stripPrefixRegex",
		"addPrefix",
		"redirectRegex",
		"redirectScheme",
		"replacePath",
		"replacePathRegex",
		"buffering",
		"circuitBreaker",
		"compress",
		"contentType",
		"retry",
		"chain",
		"plugin",
		"errors",
		"grpcWeb",
		"inFlightReq",
		"passTLSClientCert",
	}

	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	for _, mwType := range validTypes {
		t.Run(mwType, func(t *testing.T) {
			body := bytes.NewBufferString(`{
				"name": "test-` + mwType + `",
				"type": "` + mwType + `",
				"config": {}
			}`)

			c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares", body)
			handler.CreateMiddleware(c)

			if rec.Code != http.StatusCreated {
				t.Errorf("expected 201 for type %s, got %d: %s", mwType, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMiddlewareHandler_ExportMiddlewares(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config, created_at)
		VALUES ('exp-1', 'rate-limiter', 'rateLimit', '{"average":100}', '2025-01-01T00:00:00Z')
	`)
	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config, created_at)
		VALUES ('exp-2', 'my-headers', 'headers', '{"customRequestHeaders":{"X-Test":"yes"}}', '2025-01-02T00:00:00Z')
	`)
	testutil.MustExec(t, db, `
		INSERT INTO resources (id, host, service_id, org_id, site_id)
		VALUES ('res-1', 'example.com', 'svc-1', 'org-1', 'site-1')
	`)
	testutil.MustExec(t, db, `
		INSERT INTO resource_middlewares (resource_id, middleware_id, priority)
		VALUES ('res-1', 'exp-1', 50)
	`)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/export", nil)
	handler.ExportMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var snapshot map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("failed to parse export response: %v", err)
	}

	if snapshot["version"] != "1.0" {
		t.Errorf("expected version 1.0, got %v", snapshot["version"])
	}
	if snapshot["exported_at"] == nil {
		t.Error("expected exported_at field")
	}

	middlewares, ok := snapshot["middlewares"].([]interface{})
	if !ok {
		t.Fatal("expected middlewares to be an array")
	}
	if len(middlewares) != 2 {
		t.Errorf("expected 2 middlewares in export, got %d", len(middlewares))
	}

	for _, mw := range middlewares {
		m := mw.(map[string]interface{})
		if m["name"] == nil || m["type"] == nil || m["config"] == nil || m["priority"] == nil || m["created_at"] == nil {
			t.Errorf("middleware export item missing required field: %+v", m)
		}
		if m["name"] == "rate-limiter" {
			priority := int(m["priority"].(float64))
			if priority != 50 {
				t.Errorf("expected priority 50 for rate-limiter, got %d", priority)
			}
		}
		if m["name"] == "my-headers" {
			priority := int(m["priority"].(float64))
			if priority != 100 {
				t.Errorf("expected default priority 100 for my-headers, got %d", priority)
			}
		}
	}
}

func TestMiddlewareHandler_ExportMiddlewares_Empty(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/export", nil)
	handler.ExportMiddlewares(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var snapshot map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &snapshot)

	middlewares, ok := snapshot["middlewares"].([]interface{})
	if !ok {
		t.Fatal("expected middlewares to be an array")
	}
	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares in export, got %d", len(middlewares))
	}
}

func TestMiddlewareHandler_ImportMiddlewares(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	importJSON := `{
		"middlewares": [
			{"name": "imported-headers", "type": "headers", "config": {"customRequestHeaders":{"X-Import":"yes"}}, "priority": 50},
			{"name": "imported-ratelimit", "type": "rateLimit", "config": {"average":100}, "priority": 100}
		]
	}`

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares/import", bytes.NewBufferString(importJSON))
	handler.ImportMiddlewares(c)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	taskID, ok := resp["task_id"].(string)
	if !ok || taskID == "" {
		t.Fatal("expected task_id in response")
	}

	time.Sleep(500 * time.Millisecond)

	statusC, statusRec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/import/"+taskID+"/status", nil)
	statusC.Params = gin.Params{{Key: "id", Value: taskID}}
	handler.GetImportStatus(statusC)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for status, got %d: %s", statusRec.Code, statusRec.Body.String())
	}

	var status map[string]interface{}
	json.Unmarshal(statusRec.Body.Bytes(), &status)

	if status["status"] != "done" {
		t.Errorf("expected status done, got %v (full: %+v)", status["status"], status)
	}

	importedIDs, ok := status["imported_ids"].([]interface{})
	if !ok {
		t.Fatal("expected imported_ids to be an array")
	}
	if len(importedIDs) != 2 {
		t.Errorf("expected 2 imported IDs, got %d", len(importedIDs))
	}
}

func TestMiddlewareHandler_ImportMiddlewares_InvalidTypeSkipped(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	importJSON := `{
		"middlewares": [
			{"name": "valid-mw", "type": "headers", "config": {"customRequestHeaders":{"X-Test":"1"}}},
			{"name": "invalid-mw", "type": "notARealType", "config": {}}
		]
	}`

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares/import", bytes.NewBufferString(importJSON))
	handler.ImportMiddlewares(c)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	taskID := resp["task_id"].(string)

	time.Sleep(500 * time.Millisecond)

	statusC, statusRec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/import/"+taskID+"/status", nil)
	statusC.Params = gin.Params{{Key: "id", Value: taskID}}
	handler.GetImportStatus(statusC)

	var status map[string]interface{}
	json.Unmarshal(statusRec.Body.Bytes(), &status)

	if status["status"] != "done" {
		t.Errorf("expected status done, got %v", status["status"])
	}

	skipped, ok := status["skipped"].([]interface{})
	if !ok {
		t.Fatal("expected skipped to be an array")
	}
	if len(skipped) != 1 || skipped[0] != "invalid-mw" {
		t.Errorf("expected skipped to contain 'invalid-mw', got %v", skipped)
	}

	importedIDs, _ := status["imported_ids"].([]interface{})
	if len(importedIDs) != 1 {
		t.Errorf("expected 1 imported ID, got %d", len(importedIDs))
	}
}

func TestMiddlewareHandler_ImportMiddlewares_InvalidConfigTypesSkipped(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	importJSON := `{
		"middlewares": [
			{"name": "bad-config", "type": "headers", "config": {"nilField": null}},
			{"name": "good-config", "type": "headers", "config": {"simple": "string"}}
		]
	}`

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares/import", bytes.NewBufferString(importJSON))
	handler.ImportMiddlewares(c)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	taskID := resp["task_id"].(string)

	time.Sleep(500 * time.Millisecond)

	statusC, statusRec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/import/"+taskID+"/status", nil)
	statusC.Params = gin.Params{{Key: "id", Value: taskID}}
	handler.GetImportStatus(statusC)

	var status map[string]interface{}
	json.Unmarshal(statusRec.Body.Bytes(), &status)

	skipped, _ := status["skipped"].([]interface{})
	if len(skipped) != 1 || skipped[0] != "bad-config" {
		t.Errorf("expected 'bad-config' in skipped, got %v", skipped)
	}
}

func TestMiddlewareHandler_ImportMiddlewares_IdempotentOverwrite(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	testutil.MustExec(t, db, `
		INSERT INTO middlewares (id, name, type, config, created_at)
		VALUES ('old-id', 'existing-mw', 'headers', '{"old":true}', '2025-01-01T00:00:00Z')
	`)

	importJSON := `{
		"middlewares": [
			{"name": "existing-mw", "type": "rateLimit", "config": {"average":50}, "priority": 10}
		]
	}`

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares/import", bytes.NewBufferString(importJSON))
	handler.ImportMiddlewares(c)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	taskID := resp["task_id"].(string)

	time.Sleep(500 * time.Millisecond)

	statusC, statusRec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/import/"+taskID+"/status", nil)
	statusC.Params = gin.Params{{Key: "id", Value: taskID}}
	handler.GetImportStatus(statusC)

	var status map[string]interface{}
	json.Unmarshal(statusRec.Body.Bytes(), &status)

	if status["status"] != "done" {
		t.Errorf("expected status done, got %v", status["status"])
	}

	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM middlewares WHERE name = 'existing-mw'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 middleware with name existing-mw, got %d", count)
	}

	var mwType string
	db.DB.QueryRow("SELECT type FROM middlewares WHERE name = 'existing-mw'").Scan(&mwType)
	if mwType != "rateLimit" {
		t.Errorf("expected overwritten type to be rateLimit, got %s", mwType)
	}
}

func TestMiddlewareHandler_ImportMiddlewares_StatusNotFound(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodGet, "/api/middlewares/import/nonexistent/status", nil)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
	handler.GetImportStatus(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestMiddlewareHandler_ImportMiddlewares_InvalidRequestBody(t *testing.T) {
	db := testutil.NewTempDB(t)
	handler := NewMiddlewareHandler(db.DB)

	c, rec := testutil.NewContext(t, http.MethodPost, "/api/middlewares/import", bytes.NewBufferString(`{}`))
	handler.ImportMiddlewares(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestValidateConfigValueTypes(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]interface{}
		valid  bool
	}{
		{"string value", map[string]interface{}{"key": "value"}, true},
		{"int value", map[string]interface{}{"key": 42}, true},
		{"float value", map[string]interface{}{"key": 3.14}, true},
		{"bool value", map[string]interface{}{"key": true}, true},
		{"nested object with valid leaves", map[string]interface{}{"key": map[string]interface{}{"nested": "val"}}, true},
		{"nested object with nil leaf", map[string]interface{}{"key": map[string]interface{}{"nested": nil}}, false},
		{"array with valid leaves", map[string]interface{}{"key": []interface{}{"a", "b"}}, true},
		{"nil value", map[string]interface{}{"key": nil}, false},
		{"mixed valid", map[string]interface{}{"a": "str", "b": 1, "c": true, "d": 2.5}, true},
		{"deeply nested valid", map[string]interface{}{"headers": map[string]interface{}{"customRequestHeaders": map[string]interface{}{"X-Test": "yes"}}}, true},
		{"array of objects", map[string]interface{}{"items": []interface{}{map[string]interface{}{"name": "test"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateConfigValueTypes(tt.config)
			if result != tt.valid {
				t.Errorf("validateConfigValueTypes() = %v, want %v", result, tt.valid)
			}
		})
	}
}
