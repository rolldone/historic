package domain

// StorageState identifies where a topic is stored, independently of work status.
type StorageState string

const (
	StorageOpen   StorageState = "open"
	StorageClosed StorageState = "closed"
)

func (state StorageState) String() string { return string(state) }

func (state StorageState) IsValid() bool {
	return state == StorageOpen || state == StorageClosed
}
