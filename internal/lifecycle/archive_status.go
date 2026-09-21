package lifecycle

import (
	"fmt"

	"historic/internal/domain"
)

// ArchiveStatus is retained for callers compiled against the old API. Work
// status changes no longer move topics between storage roots.
func (service Service) ArchiveStatus(id domain.ID, next domain.Status) (Change, error) {
	if !next.IsClose() {
		return Change{}, fmt.Errorf("%w: archive requires close status %s", domain.ErrInvalidStatus, next)
	}
	change, err := service.ChangeStatus(id, next)
	if err != nil {
		return Change{}, err
	}
	return change, nil
}
