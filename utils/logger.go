package utils

import (
	"time"

	"github.com/xxjwxc/gowp/workpool"
)

func PoolProgress(wp *workpool.WorkPool, callback func(), optionalDuration ...time.Duration) {
	every := 1 * time.Minute
	if len(optionalDuration) > 0 {
		every = optionalDuration[0]
	}

	for {
		if wp.IsDone() {
			return
		}
		callback()
		time.Sleep(every)
	}
}
