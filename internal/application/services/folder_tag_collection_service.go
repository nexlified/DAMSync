package services

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

// FolderServiceImpl implements FolderService.
type FolderServiceImpl struct {
	folderRepo outbound.FolderRepository
}

func NewFolderService(folderRepo outbound.FolderRepository) *FolderServiceImpl {
	return &FolderServiceImpl{folderRepo: folderRepo}
}

func (s *FolderServiceImpl) CreateFolder(ctx context.Context, orgID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Folder, error) {
	var path string
	if parentID != nil {
		parent, err := s.folderRepo.GetByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent.OrgID != orgID {
			return nil, domain.ErrNotFound
		}
		path = parent.Path + "/" + slugify(name)
	} else {
		path = "/" + slugify(name)
	}

	folder := &domain.Folder{
		ID:        uuid.New(),
		OrgID:     orgID,
		ParentID:  parentID,
		Name:      name,
		Path:      path,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.folderRepo.Create(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *FolderServiceImpl) GetFolder(ctx context.Context, orgID, folderID uuid.UUID) (*domain.Folder, error) {
	f, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if f.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return f, nil
}

func (s *FolderServiceImpl) UpdateFolder(ctx context.Context, orgID, folderID uuid.UUID, name string) (*domain.Folder, error) {
	f, err := s.GetFolder(ctx, orgID, folderID)
	if err != nil {
		return nil, err
	}
	f.Name = name
	f.UpdatedAt = time.Now().UTC()
	if err := s.folderRepo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *FolderServiceImpl) DeleteFolder(ctx context.Context, orgID, folderID uuid.UUID) error {
	f, err := s.GetFolder(ctx, orgID, folderID)
	if err != nil {
		return err
	}
	return s.folderRepo.Delete(ctx, f.ID)
}

func (s *FolderServiceImpl) ListFolders(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error) {
	return s.folderRepo.ListByOrg(ctx, orgID)
}

func (s *FolderServiceImpl) GetFolderTree(ctx context.Context, orgID uuid.UUID) ([]*inbound.FolderNode, error) {
	all, err := s.folderRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return buildFolderTree(all, nil), nil
}

func buildFolderTree(all []*domain.Folder, parentID *uuid.UUID) []*inbound.FolderNode {
	nodes := make([]*inbound.FolderNode, 0)
	for _, f := range all {
		if (parentID == nil && f.ParentID == nil) ||
			(parentID != nil && f.ParentID != nil && *f.ParentID == *parentID) {
			node := &inbound.FolderNode{
				Folder:   f,
				Children: buildFolderTree(all, &f.ID),
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// TagServiceImpl implements TagService.
type TagServiceImpl struct {
	tagRepo   outbound.TagRepository
	assetRepo outbound.AssetRepository
}

func NewTagService(tagRepo outbound.TagRepository, assetRepo outbound.AssetRepository) *TagServiceImpl {
	return &TagServiceImpl{tagRepo: tagRepo, assetRepo: assetRepo}
}

func (s *TagServiceImpl) CreateTag(ctx context.Context, orgID uuid.UUID, name string) (*domain.Tag, error) {
	tag := &domain.Tag{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Slug:      slugify(name),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *TagServiceImpl) ListTags(ctx context.Context, orgID uuid.UUID) ([]*domain.Tag, error) {
	return s.tagRepo.ListByOrg(ctx, orgID)
}

func (s *TagServiceImpl) DeleteTag(ctx context.Context, orgID, tagID uuid.UUID) error {
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return err
	}
	if tag.OrgID != orgID {
		return domain.ErrNotFound
	}
	return s.tagRepo.Delete(ctx, tagID)
}

func (s *TagServiceImpl) TagAsset(ctx context.Context, orgID, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if asset.OrgID != orgID {
		return domain.ErrNotFound
	}
	return s.assetRepo.AddTags(ctx, assetID, tagIDs)
}

func (s *TagServiceImpl) UntagAsset(ctx context.Context, orgID, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return err
	}
	if asset.OrgID != orgID {
		return domain.ErrNotFound
	}
	return s.assetRepo.RemoveTags(ctx, assetID, tagIDs)
}

// CollectionServiceImpl implements CollectionService.
type CollectionServiceImpl struct {
	collectionRepo outbound.CollectionRepository
}

func NewCollectionService(collectionRepo outbound.CollectionRepository) *CollectionServiceImpl {
	return &CollectionServiceImpl{collectionRepo: collectionRepo}
}

func (s *CollectionServiceImpl) CreateCollection(ctx context.Context, orgID uuid.UUID, name, description string) (*domain.Collection, error) {
	c := &domain.Collection{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := s.collectionRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CollectionServiceImpl) GetCollection(ctx context.Context, orgID, collectionID uuid.UUID) (*domain.Collection, error) {
	c, err := s.collectionRepo.GetByID(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if c.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (s *CollectionServiceImpl) UpdateCollection(ctx context.Context, orgID, collectionID uuid.UUID, name, description string) (*domain.Collection, error) {
	c, err := s.GetCollection(ctx, orgID, collectionID)
	if err != nil {
		return nil, err
	}
	c.Name = name
	c.Description = description
	c.UpdatedAt = time.Now().UTC()
	if err := s.collectionRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CollectionServiceImpl) DeleteCollection(ctx context.Context, orgID, collectionID uuid.UUID) error {
	c, err := s.GetCollection(ctx, orgID, collectionID)
	if err != nil {
		return err
	}
	return s.collectionRepo.Delete(ctx, c.ID)
}

func (s *CollectionServiceImpl) ListCollections(ctx context.Context, orgID uuid.UUID) ([]*domain.Collection, error) {
	return s.collectionRepo.ListByOrg(ctx, orgID)
}

func (s *CollectionServiceImpl) AddAsset(ctx context.Context, orgID, collectionID, assetID uuid.UUID) error {
	if _, err := s.GetCollection(ctx, orgID, collectionID); err != nil {
		return err
	}
	return s.collectionRepo.AddAsset(ctx, collectionID, assetID, 0)
}

func (s *CollectionServiceImpl) RemoveAsset(ctx context.Context, orgID, collectionID, assetID uuid.UUID) error {
	if _, err := s.GetCollection(ctx, orgID, collectionID); err != nil {
		return err
	}
	return s.collectionRepo.RemoveAsset(ctx, collectionID, assetID)
}

func (s *CollectionServiceImpl) ListAssets(ctx context.Context, orgID, collectionID uuid.UUID, limit, offset int) ([]*domain.Asset, error) {
	if _, err := s.GetCollection(ctx, orgID, collectionID); err != nil {
		return nil, err
	}
	return s.collectionRepo.ListAssets(ctx, collectionID, limit, offset)
}

// SearchServiceImpl implements SearchService.
type SearchServiceImpl struct {
	assetRepo outbound.AssetRepository
}

func NewSearchService(assetRepo outbound.AssetRepository) *SearchServiceImpl {
	return &SearchServiceImpl{assetRepo: assetRepo}
}

func (s *SearchServiceImpl) Search(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, int, error) {
	assets, cursor, err := s.assetRepo.List(ctx, filter)
	if err != nil {
		return nil, "", 0, err
	}
	count, err := s.assetRepo.Count(ctx, filter)
	if err != nil {
		return assets, cursor, 0, nil
	}
	return assets, cursor, count, nil
}

// --- helpers ---

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
