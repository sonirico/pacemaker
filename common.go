package pacemaker

import "time"

type clock interface {
	Now() time.Time
}
