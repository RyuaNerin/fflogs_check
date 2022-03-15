package analysispool

import (
	"time"

	"ffxiv_check/cache"
)

var (
	csTemplate = cache.NewStorage(
		"./_cachedata/results",
		time.Hour,
		"./analysis/",
		"./ffxiv",
	)
)
