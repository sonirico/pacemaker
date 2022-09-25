package pacemaker

import "time"

type (
	Result struct {
		TimeToWait time.Duration
		FreeSlots  int64
	}
)

var (
	nores = Result{}
)

func res(ttw time.Duration, slots int64) Result {
	return Result{TimeToWait: ttw, FreeSlots: slots}
}
