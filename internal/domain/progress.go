package domain

// ProgressPercent derives the topic progress percentage from the statuses of
// its managed files. It is a derived read model, never persisted to Markdown.
//
// Rules:
//   - no managed files -> nil (no percentage)
//   - close_count = files whose status IsClose()
//   - percent = floor(100 * close_count / managed_count), clamped by construction
//
// Asset files must not be included by callers. Each managed file carries the
// same flat weight, so one file equals one point.
func ProgressPercent(statuses []Status) *int {
	if len(statuses) == 0 {
		return nil
	}
	closeCount := 0
	for _, status := range statuses {
		if status.IsClose() {
			closeCount++
		}
	}
	percent := (100 * closeCount) / len(statuses)
	return &percent
}
