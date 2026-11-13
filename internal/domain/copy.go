package domain

// Copy regions describe where a duplicated topic lives. The open workdir holds
// the "local" copy the user edits; `.historic/.database/` holds the "database"
// mirror that travels through pull/push.
const (
	// CopyRegionLocal is the open workdir copy.
	CopyRegionLocal = "local"
	// CopyRegionDatabase is the closed mirror copy under `.historic/.database/`.
	CopyRegionDatabase = "database"
)

// CopyRegionCaption is the short human notice printed on the newer copy of a
// duplicated topic. It answers "which side moved" in one read, which is also
// why the save guard will refuse the write.
func CopyRegionCaption(region string) string {
	switch region {
	case CopyRegionDatabase:
		return "newer in database"
	case CopyRegionLocal:
		return "newer locally"
	default:
		return ""
	}
}

// CopyRegionForStorage maps a storage scope onto its copy region name.
func CopyRegionForStorage(storage string) string {
	if storage == StorageOpen.String() {
		return CopyRegionLocal
	}
	return CopyRegionDatabase
}
