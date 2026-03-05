package v1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/domain"
	"github.com/nexlified/dam/internal/infrastructure/http/middleware"
	"github.com/nexlified/dam/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func testErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"
	if fe, ok := err.(*fiber.Error); ok {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	if err != nil {
		s := err.Error()
		switch {
		case strings.Contains(s, "not found"):
			code, msg = fiber.StatusNotFound, s
		case strings.Contains(s, "unauthorized"):
			code, msg = fiber.StatusUnauthorized, "unauthorized"
		case strings.Contains(s, "invalid"):
			code, msg = fiber.StatusBadRequest, s
		}
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}

// injectOrg sets org_id in Fiber locals, simulating successful auth.
func injectOrg(orgID uuid.UUID) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(middleware.ContextKeyOrgID, orgID)
		return c.Next()
	}
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{ErrorHandler: testErrorHandler})
}

func doRequest(app *fiber.App, method, path, body string) *http.Response {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return b
}

// ── GET /folders/tree ─────────────────────────────────────────────────────────

func TestFolderTree_EmptyOrg_Returns200(t *testing.T) {
	orgID := uuid.New()
	svc := &testutil.MockFolderService{} // GetFolderTreeFn is nil → returns nil, nil
	handler := NewFoldersHandler(svc)

	app := newTestApp()
	app.Get("/folders/tree", injectOrg(orgID), handler.Tree)

	resp := doRequest(app, "GET", "/folders/tree", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	assert.Contains(t, result, "data", "response must have 'data' key")
}

func TestFolderTree_WithFolders_Returns200(t *testing.T) {
	orgID := uuid.New()
	folderID := uuid.New()
	folder := testutil.NewFolder(folderID, orgID, "photos")

	svc := &testutil.MockFolderService{
		GetFolderTreeFn: func(ctx context.Context, id uuid.UUID) ([]*inbound.FolderNode, error) {
			return []*inbound.FolderNode{
				{Folder: folder, Children: []*inbound.FolderNode{}},
			}, nil
		},
	}
	handler := NewFoldersHandler(svc)

	app := newTestApp()
	app.Get("/folders/tree", injectOrg(orgID), handler.Tree)

	resp := doRequest(app, "GET", "/folders/tree", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	assert.True(t, strings.HasPrefix(string(result["data"]), "["), "data must be a JSON array")
}

// ── FolderNode JSON shape ─────────────────────────────────────────────────────

func TestFolderNode_JSONShape_FlatSnakeCase(t *testing.T) {
	// Regression: FolderNode used to serialise as {"Folder":{...}, "Children":[...]}
	// instead of flat {"id":..., "name":..., "children":[...]}.
	orgID := uuid.New()
	folder := testutil.NewFolder(uuid.New(), orgID, "photos")

	node := &inbound.FolderNode{
		Folder:   folder,
		Children: []*inbound.FolderNode{},
	}

	data, err := json.Marshal(node)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	assert.Contains(t, m, "id", "JSON must contain 'id'")
	assert.Contains(t, m, "name", "JSON must contain 'name'")
	assert.Contains(t, m, "org_id", "JSON must contain 'org_id'")
	assert.Contains(t, m, "children", "JSON must contain 'children'")
	assert.NotContains(t, m, "Folder", "must NOT have nested 'Folder' wrapper")
	assert.NotContains(t, m, "Children", "'children' must be lowercase")
	assert.NotContains(t, m, "ID", "field names must be snake_case not PascalCase")
}

// ── GET /assets ───────────────────────────────────────────────────────────────

func TestAssetList_Returns200WithDataAndCursor(t *testing.T) {
	orgID := uuid.New()
	handler := NewAssetsHandler(&testutil.MockAssetService{}, &testutil.MockStoragePort{})

	app := newTestApp()
	app.Get("/assets", injectOrg(orgID), handler.List)

	resp := doRequest(app, "GET", "/assets", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	assert.Contains(t, result, "data", "response must have 'data' key")
	assert.Contains(t, result, "next_cursor", "response must have 'next_cursor' key")
}

func TestAssetList_DataIsAlwaysArray(t *testing.T) {
	// Even when the service returns nil assets, data must serialise as [] not null.
	orgID := uuid.New()
	handler := NewAssetsHandler(&testutil.MockAssetService{}, &testutil.MockStoragePort{})

	app := newTestApp()
	app.Get("/assets", injectOrg(orgID), handler.List)

	resp := doRequest(app, "GET", "/assets", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	assert.True(t, strings.HasPrefix(string(result["data"]), "["),
		"data must be a JSON array, got: %s", string(result["data"]))
}

func TestAssetList_WithAssets_ReturnsEnrichedFields(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	asset := testutil.NewAsset(assetID, orgID)

	svc := &testutil.MockAssetService{
		ListAssetsFn: func(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error) {
			return []*domain.Asset{asset}, "next-cursor", nil
		},
	}
	handler := NewAssetsHandler(svc, &testutil.MockStoragePort{})

	app := newTestApp()
	app.Get("/assets", injectOrg(orgID), handler.List)

	resp := doRequest(app, "GET", "/assets", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Data       []map[string]interface{} `json:"data"`
		NextCursor string                   `json:"next_cursor"`
	}
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	require.Len(t, result.Data, 1)
	assert.Equal(t, "next-cursor", result.NextCursor)

	a := result.Data[0]
	assert.Contains(t, a, "id")
	assert.Contains(t, a, "filename")
	assert.Contains(t, a, "url", "enriched response must include 'url' field")
}

func TestAssetList_SortByPassedToService(t *testing.T) {
	orgID := uuid.New()
	var capturedFilter domain.AssetListFilter

	svc := &testutil.MockAssetService{
		ListAssetsFn: func(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error) {
			capturedFilter = filter
			return nil, "", nil
		},
	}
	handler := NewAssetsHandler(svc, &testutil.MockStoragePort{})

	app := newTestApp()
	app.Get("/assets", injectOrg(orgID), handler.List)

	resp := doRequest(app, "GET", "/assets?sort_by=filename&sort_dir=asc", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "filename", capturedFilter.SortBy)
	assert.Equal(t, "asc", capturedFilter.SortDir)
}

// ── POST /assets/:id/move ─────────────────────────────────────────────────────

func TestAssetMove_InvalidUUID_Returns400(t *testing.T) {
	handler := NewAssetsHandler(&testutil.MockAssetService{}, &testutil.MockStoragePort{})
	app := newTestApp()
	app.Post("/assets/:id/move", injectOrg(uuid.New()), handler.Move)

	resp := doRequest(app, "POST", "/assets/not-a-uuid/move", `{"folder_id":null}`)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ── DELETE /assets/:id ────────────────────────────────────────────────────────

func TestAssetDelete_Success_Returns204(t *testing.T) {
	orgID := uuid.New()
	handler := NewAssetsHandler(&testutil.MockAssetService{}, &testutil.MockStoragePort{})
	app := newTestApp()
	app.Delete("/assets/:id", injectOrg(orgID), handler.Delete)

	resp := doRequest(app, "DELETE", "/assets/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestAssetDelete_NotFound_Returns404(t *testing.T) {
	orgID := uuid.New()
	svc := &testutil.MockAssetService{
		DeleteAssetFn: func(ctx context.Context, orgID, assetID uuid.UUID) error {
			return domain.ErrNotFound
		},
	}
	handler := NewAssetsHandler(svc, &testutil.MockStoragePort{})
	app := newTestApp()
	app.Delete("/assets/:id", injectOrg(orgID), handler.Delete)

	resp := doRequest(app, "DELETE", "/assets/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ── GET /assets/:id/signed-url ────────────────────────────────────────────────

func TestGetSignedURL_Returns200WithURLAndExpiry(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()

	svc := &testutil.MockAssetService{
		GenerateSignedURLFn: func(ctx context.Context, o, a uuid.UUID, ttl time.Duration) (string, error) {
			return "/secure/abc123/9999999999/orgs/x/assets/y.jpg", nil
		},
	}
	handler := NewAssetsHandler(svc, &testutil.MockStoragePort{})
	app := newTestApp()
	app.Get("/assets/:id/signed-url", injectOrg(orgID), handler.GetSignedURL)

	resp := doRequest(app, "GET", "/assets/"+assetID.String()+"/signed-url", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(readBody(t, resp), &result))
	assert.Contains(t, result, "url")
	assert.Contains(t, result, "expires_in")
}

func TestGetSignedURL_InvalidUUID_Returns400(t *testing.T) {
	handler := NewAssetsHandler(&testutil.MockAssetService{}, &testutil.MockStoragePort{})
	app := newTestApp()
	app.Get("/assets/:id/signed-url", injectOrg(uuid.New()), handler.GetSignedURL)

	resp := doRequest(app, "GET", "/assets/not-a-uuid/signed-url", "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ── POST /folders ─────────────────────────────────────────────────────────────

func TestCreateFolder_MissingName_Returns400(t *testing.T) {
	handler := NewFoldersHandler(&testutil.MockFolderService{})
	app := newTestApp()
	app.Post("/folders", injectOrg(uuid.New()), handler.Create)

	resp := doRequest(app, "POST", "/folders", `{"name":""}`)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateFolder_Success_Returns201(t *testing.T) {
	orgID := uuid.New()
	handler := NewFoldersHandler(&testutil.MockFolderService{})
	app := newTestApp()
	app.Post("/folders", injectOrg(orgID), handler.Create)

	resp := doRequest(app, "POST", "/folders", `{"name":"My Folder"}`)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

// ── DELETE /folders/:id ───────────────────────────────────────────────────────

func TestDeleteFolder_NotFound_Returns404(t *testing.T) {
	orgID := uuid.New()
	svc := &testutil.MockFolderService{
		DeleteFolderFn: func(ctx context.Context, o, f uuid.UUID) error {
			return domain.ErrNotFound
		},
	}
	handler := NewFoldersHandler(svc)
	app := newTestApp()
	app.Delete("/folders/:id", injectOrg(orgID), handler.Delete)

	resp := doRequest(app, "DELETE", "/folders/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
