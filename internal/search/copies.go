package search

import (
	"historic/internal/display"
	"historic/internal/domain"
)

// CollapseIdenticalCopies folds mirrored copies that share a freshness stamp
// into a single row — the open ("local") copy — and flags it Mirrored. Copies
// whose stamps differ keep their own row, because then the duplicate is the
// signal that one side moved ahead; annotateResultCopies names the newer one.
//
// Both storage copies are always read from the index; this function only decides
// what a listing surface shows. It runs before the page is cut so pagination
// metadata reflects the rows the caller receives.
func CollapseIdenticalCopies(results []Result) []Result {
	if len(results) < 2 {
		return results
	}
	rows := make([]display.CopyRow, len(results))
	for index, result := range results {
		rows[index] = display.CopyRow{
			Key:   CopyKey(result),
			Stamp: result.UpdatedAt,
			Open:  result.Storage == domain.StorageOpen.String(),
		}
	}
	kept := display.CollapseIdenticalCopies(rows)
	collapsed := make([]Result, 0, len(kept))
	for _, row := range kept {
		result := results[row.Index]
		result.Mirrored = row.Mirrored
		collapsed = append(collapsed, result)
	}
	return collapsed
}
