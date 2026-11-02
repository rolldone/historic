package domain

import "sort"

// openStatusPriority is the deterministic priority order for open statuses.
// Higher-priority (earlier) statuses win when a topic still has open work.
var openStatusPriority = []Status{
	StatusBlocked,
	StatusReview,
	StatusProgress,
	StatusPlanned,
	StatusPending,
	StatusDraft,
	StatusCreate,
}

// AggregateStatus derives a single topic status from the statuses of its
// managed files. It is deterministic, idempotent, and uses priority ordering
// rather than arithmetic.
//
// Rules:
//   - no managed files -> empty status
//   - any open file -> highest-priority open status
//   - otherwise, failed wins, then all-cancelled, then complete
//
// forwarded and archived are treated like complete for aggregation; asset
// files must not be included by callers.
func AggregateStatus(statuses []Status) Status {
	if len(statuses) == 0 {
		return ""
	}
	present := make(map[Status]struct{}, len(statuses))
	for _, status := range statuses {
		present[status] = struct{}{}
	}
	for _, candidate := range openStatusPriority {
		if _, ok := present[candidate]; ok {
			return candidate
		}
	}
	if _, ok := present[StatusFailed]; ok {
		return StatusFailed
	}
	allCancelled := true
	for _, status := range statuses {
		if status != StatusCancelled {
			allCancelled = false
			break
		}
	}
	if allCancelled {
		return StatusCancelled
	}
	return StatusComplete
}

// OpenStatuses returns the registered open statuses in deterministic order.
func OpenStatuses() []Status {
	return sortedStatuses(openStatuses)
}

// CloseStatuses returns the registered close statuses in deterministic order.
func CloseStatuses() []Status {
	return sortedStatuses(closeStatuses)
}

// ValidStatuses returns every registered lifecycle status in deterministic order.
func ValidStatuses() []Status {
	return sortedStatuses(validStatuses)
}

func sortedStatuses(source map[Status]struct{}) []Status {
	result := make([]Status, 0, len(source))
	for status := range source {
		result = append(result, status)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
