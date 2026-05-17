package httpx

// jsonSlice ensures JSON encodes empty lists as [] not null.
func jsonSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
