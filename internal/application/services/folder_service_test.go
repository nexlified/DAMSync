package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/nexlified/dam/domain"
	"github.com/nexlified/dam/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFolderSvc(repo *testutil.MockFolderRepository) *FolderServiceImpl {
	return NewFolderService(repo)
}

// ── GetFolderTree ─────────────────────────────────────────────────────────────

func TestGetFolderTree_EmptyOrg_ReturnsEmptySlice(t *testing.T) {
	// When the org has no folders, ListByOrg returns nil.
	// buildFolderTree must return an empty (non-nil) slice so JSON encodes as []
	// not null — which would cause a frontend TypeError on .map().
	repo := &testutil.MockFolderRepository{
		ListByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Folder, error) {
			return nil, nil
		},
	}
	svc := newFolderSvc(repo)

	tree, err := svc.GetFolderTree(context.Background(), uuid.New())
	require.NoError(t, err)

	// Marshalling nil slice produces "null"; empty slice produces "[]"
	data, _ := json.Marshal(tree)
	assert.JSONEq(t, "[]", string(data), "empty tree must serialise as [] not null")
}

func TestGetFolderTree_FlatFolders(t *testing.T) {
	orgID := uuid.New()
	f1 := testutil.NewFolder(uuid.New(), orgID, "photos")
	f2 := testutil.NewFolder(uuid.New(), orgID, "docs")

	repo := &testutil.MockFolderRepository{
		ListByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Folder, error) {
			return []*domain.Folder{f1, f2}, nil
		},
	}
	svc := newFolderSvc(repo)

	tree, err := svc.GetFolderTree(context.Background(), orgID)
	require.NoError(t, err)
	assert.Len(t, tree, 2)
}

func TestGetFolderTree_NestedFolders(t *testing.T) {
	orgID := uuid.New()
	parent := testutil.NewFolder(uuid.New(), orgID, "photos")
	child := testutil.NewFolder(uuid.New(), orgID, "2024")
	child.ParentID = &parent.ID

	repo := &testutil.MockFolderRepository{
		ListByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Folder, error) {
			return []*domain.Folder{parent, child}, nil
		},
	}
	svc := newFolderSvc(repo)

	tree, err := svc.GetFolderTree(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, tree, 1, "only one root node expected")
	assert.Equal(t, parent.ID, tree[0].ID)
	require.Len(t, tree[0].Children, 1, "parent must have one child")
	assert.Equal(t, child.ID, tree[0].Children[0].ID)
}

// ── FolderNode JSON shape ─────────────────────────────────────────────────────

func TestFolderNode_JSONShape_FlatSnakeCase(t *testing.T) {
	// This is the regression test for the bug where FolderNode serialised as
	// {"Folder": {"ID": ...}, "Children": [...]} instead of flat snake_case.
	// The frontend Folder type expects: id, org_id, parent_id, name, path, children.
	orgID := uuid.New()
	folder := testutil.NewFolder(uuid.New(), orgID, "photos")

	repo := &testutil.MockFolderRepository{
		ListByOrgFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Folder, error) {
			return []*domain.Folder{folder}, nil
		},
	}
	svc := newFolderSvc(repo)
	tree, err := svc.GetFolderTree(context.Background(), orgID)
	require.NoError(t, err)
	require.Len(t, tree, 1)

	data, err := json.Marshal(tree[0])
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))

	// Must have flat snake_case fields — not a nested "Folder" wrapper
	assert.Contains(t, m, "id", "JSON must contain 'id' field")
	assert.Contains(t, m, "name", "JSON must contain 'name' field")
	assert.Contains(t, m, "org_id", "JSON must contain 'org_id' field")
	assert.Contains(t, m, "children", "JSON must contain 'children' field")
	assert.NotContains(t, m, "Folder", "JSON must NOT have a nested 'Folder' wrapper")
	assert.NotContains(t, m, "ID", "field names must be snake_case, not PascalCase")
}

// ── CreateFolder ──────────────────────────────────────────────────────────────

func TestCreateFolder_RootFolder(t *testing.T) {
	orgID := uuid.New()
	var created *domain.Folder
	repo := &testutil.MockFolderRepository{
		CreateFn: func(_ context.Context, f *domain.Folder) error {
			created = f
			return nil
		},
	}
	svc := newFolderSvc(repo)

	folder, err := svc.CreateFolder(context.Background(), orgID, nil, "My Folder")
	require.NoError(t, err)
	assert.Equal(t, "My Folder", folder.Name)
	assert.Equal(t, orgID, folder.OrgID)
	assert.Nil(t, folder.ParentID)
	require.NotNil(t, created)
}

func TestCreateFolder_WithParent(t *testing.T) {
	orgID := uuid.New()
	parentID := uuid.New()
	parent := testutil.NewFolder(parentID, orgID, "photos")

	repo := &testutil.MockFolderRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Folder, error) {
			if id == parentID {
				return parent, nil
			}
			return nil, domain.ErrNotFound
		},
		CreateFn: func(_ context.Context, _ *domain.Folder) error { return nil },
	}
	svc := newFolderSvc(repo)

	folder, err := svc.CreateFolder(context.Background(), orgID, &parentID, "2024")
	require.NoError(t, err)
	require.NotNil(t, folder.ParentID)
	assert.Equal(t, parentID, *folder.ParentID)
}

func TestCreateFolder_ParentNotFound(t *testing.T) {
	orgID := uuid.New()
	nonExistentParent := uuid.New()

	repo := &testutil.MockFolderRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Folder, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := newFolderSvc(repo)

	_, err := svc.CreateFolder(context.Background(), orgID, &nonExistentParent, "child")
	assert.Error(t, err)
}

// ── DeleteFolder ──────────────────────────────────────────────────────────────

func TestDeleteFolder_NotFound(t *testing.T) {
	repo := &testutil.MockFolderRepository{}
	svc := newFolderSvc(repo)

	err := svc.DeleteFolder(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestDeleteFolder_WrongOrg(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	folderID := uuid.New()

	repo := &testutil.MockFolderRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Folder, error) {
			return testutil.NewFolder(folderID, otherOrgID, "photos"), nil
		},
	}
	svc := newFolderSvc(repo)

	err := svc.DeleteFolder(context.Background(), orgID, folderID)
	assert.Error(t, err, "deleting another org's folder must fail")
}
