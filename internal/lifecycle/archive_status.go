package lifecycle

import (
	"fmt"

	"historic/internal/domain"
)

// ArchiveStatus is retained for callers compiled against the old API. Topic
// work status is not authoritative; storage is controlled by open/close.
func (service Service) ArchiveStatus(id domain.ID, next domain.Status) (Change, error) {
	return Change{}, fmt.Errorf("%w: topic has no work status; use file status and close/open", domain.ErrConflict)
}
