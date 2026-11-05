package firestore

import "errors"

var (
	// ErrRepositoryNotFound reports that an aggregate document does not exist.
	ErrRepositoryNotFound = errors.New("repository aggregate not found")
	// ErrRepositoryConflict reports a duplicate create or stale aggregate write.
	ErrRepositoryConflict = errors.New("repository aggregate revision conflict")
	// ErrConcurrentUpdate reports an optimistic-concurrency precondition failure.
	ErrConcurrentUpdate = errors.New("repository aggregate was concurrently updated")
	// ErrRepositoryCorrupt reports persisted state that cannot be rehydrated.
	ErrRepositoryCorrupt = errors.New("repository aggregate state is corrupt")
	// ErrRepositoryUnavailable reports an infrastructure failure while accessing persistence.
	ErrRepositoryUnavailable = errors.New("repository infrastructure unavailable")
)
