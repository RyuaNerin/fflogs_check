package share

import "sort"

func StringInSortedSlice(a []string, x string) bool {
	idx := sort.SearchStrings(a, x)
	if idx >= len(a) {
		return false
	}

	return a[idx] == x
}
