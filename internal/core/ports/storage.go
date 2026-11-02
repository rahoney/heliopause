package ports

import (
	"context"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// StorageRequest identifies controller-created storage using opaque root and
// relative handles. Concrete owners retain creation, publication and deletion.
type StorageRequest struct {
	Operation         domain.OperationID
	Run               domain.RunID
	Kind              string
	Temporary, Target string
	Bytes, Files      uint64
	// Extend reserves additional writes to this operation's already sealed intake.
	// Bytes/Files are additional bounds; original bytes remain charged.
	Extend bool
}

// Storage accounts a reservation durably before the owner writes payloads.
type Storage interface {
	Policy() domain.StoragePolicy
	Reserve(context.Context, StorageRequest) (StorageLease, error)
}

type StorageLease interface {
	Commit(context.Context) error
	// Abort releases accounting only after the owner has removed both names.
	// Failure retains the reservation for recovery; Close/PID absence is no proof.
	Abort(context.Context) error
	// Close releases descriptors, preserving all durable accounting.
	Close() error
}

// InspectionStorage records a trusted caller's input-specific completed-check
// boundary, including static original-layout proofs and supporting inspection
// inputs. It never approves installation; outer finalization waits for all
// shared consumers and their guards/runtime closes.
type InspectionStorage interface {
	NoteInspection(context.Context, domain.RunID, bool) error
}

type storageContextKey struct{}
type storageOperationKey struct{}

func WithStorage(ctx context.Context, storage Storage) context.Context {
	return context.WithValue(ctx, storageContextKey{}, storage)
}
func StorageFromContext(ctx context.Context) Storage {
	if ctx == nil {
		return nil
	}
	storage, _ := ctx.Value(storageContextKey{}).(Storage)
	return storage
}
func WithStorageOperation(ctx context.Context, operation domain.OperationID) context.Context {
	return context.WithValue(ctx, storageOperationKey{}, operation)
}
func StorageOperationFromContext(ctx context.Context) domain.OperationID {
	if ctx == nil {
		return domain.OperationID{}
	}
	operation, _ := ctx.Value(storageOperationKey{}).(domain.OperationID)
	return operation
}
