package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/dbdriver"

	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// DatabaseService orchestrates logical database operations across instances.
type DatabaseService struct {
	manager      *DatabaseManager
	databaseRepo *dbrepo.DatabaseRepository
}

// NewDatabaseService constructs a DatabaseService.
func NewDatabaseService(manager *DatabaseManager, databaseRepo *dbrepo.DatabaseRepository) *DatabaseService {
	return &DatabaseService{
		manager:      manager,
		databaseRepo: databaseRepo,
	}
}

// CreateDatabaseInput captures options for creating a database.
type CreateDatabaseInput struct {
	Name      string
	Charset   string
	Collation string
	Owner     string
}

// CreateDatabase creates a logical database and persists metadata.
func (s *DatabaseService) CreateDatabase(ctx context.Context, instanceID uuid.UUID, input CreateDatabaseInput) (*models.Database, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("database name is required")
	}

	isUnique, err := s.databaseRepo.CheckUniqueName(ctx, instanceID, input.Name)
	if err != nil {
		return nil, err
	}
	if !isUnique {
		return nil, fmt.Errorf("database %s already exists", input.Name)
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return nil, ErrOperationNotSupported
	}

	opts := dbdriver.CreateDatabaseOptions{
		Name:      input.Name,
		Charset:   input.Charset,
		Collation: input.Collation,
		Owner:     input.Owner,
	}

	if err := driver.CreateDatabase(ctx, opts); err != nil {
		return nil, err
	}

	record := &models.Database{
		InstanceID: instanceID,
		Name:       input.Name,
		Charset:    input.Charset,
		Collation:  input.Collation,
		Owner:      input.Owner,
	}

	if err := s.databaseRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// ListDatabases returns databases for an instance merging remote metadata with local records.
func (s *DatabaseService) ListDatabases(ctx context.Context, instanceID uuid.UUID) ([]*models.Database, error) {
	driver, _, err := s.manager.GetConnectedDriver(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	remote, err := driver.ListDatabases(ctx)
	if err != nil {
		return nil, err
	}

	localList, err := s.databaseRepo.ListByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	localMap := make(map[string]*models.Database, len(localList))
	for _, db := range localList {
		localMap[strings.ToLower(db.Name)] = db
	}

	results := make([]*models.Database, 0, len(remote)+len(localList))
	for _, info := range remote {
		key := strings.ToLower(info.Name)
		if existing, ok := localMap[key]; ok {
			existing.Charset = info.Charset
			existing.Collation = info.Collation
			if info.Owner != "" {
				existing.Owner = info.Owner
			}
			existing.SizeBytes = info.SizeBytes
			existing.TableCount = info.TableCount
			if info.KeyCount > 0 {
				existing.KeyCount = info.KeyCount
			}
			results = append(results, existing)
			delete(localMap, key)
		} else {
			results = append(results, &models.Database{
				InstanceID: instanceID,
				Name:       info.Name,
				Charset:    info.Charset,
				Collation:  info.Collation,
				Owner:      info.Owner,
				SizeBytes:  info.SizeBytes,
				TableCount: info.TableCount,
				KeyCount:   info.KeyCount,
			})
		}
	}

	// Append any remaining local records (may represent unmanaged databases).
	for _, leftover := range localMap {
		results = append(results, leftover)
	}

	return results, nil
}

// DeleteDatabase removes a database remotely and deletes metadata.
func (s *DatabaseService) DeleteDatabase(ctx context.Context, databaseID uuid.UUID, confirmName string) (*models.Database, error) {
	dbRecord, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if confirmName != "" && !strings.EqualFold(confirmName, dbRecord.Name) {
		return nil, fmt.Errorf("confirmation name mismatch")
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, dbRecord.InstanceID)
	if err != nil {
		return nil, err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return nil, ErrOperationNotSupported
	}

	opts := map[string]interface{}{
		"confirm_name": dbRecord.Name,
	}
	if err := driver.DeleteDatabase(ctx, dbRecord.Name, opts); err != nil {
		return nil, err
	}

	if err := s.databaseRepo.Delete(ctx, databaseID); err != nil {
		return nil, err
	}

	return dbRecord, nil
}
