package pacemaker

//import (
//	"sync"
//	"time"
//)
//
//func windowFuncFactory(args FixedWindowArgs) func(time.Time) time.Time {
//	if args.TruncateWindow {
//		return func(t time.Time) time.Time {
//			return t.Truncate(args.Rate.Duration())
//		}
//	}
//
//	return func(t time.Time) time.Time {
//		return t.Add(args.Rate.Duration())
//	}
//}
//
//type FixedWindowArgs struct {
//	Capacity int
//	Rate     Rate
//	// TruncateWindow indicates whether the time window in created by truncating the first request time of arrival or if
//	// the first request initiates the window as is.
//	TruncateWindow bool
//}
//
//type FixedWindowRateLimiter struct {
//	rate Rate
//
//	clock RealClock
//
//	window time.Time
//
//	mu sync.Mutex
//
//	capacity     int
//	overCapacity bool
//}
//
//func (l *FixedWindowRateLimiter) Check(_ FixedWindowArgs) (ttw time.Duration, pass bool) {
//	l.mu.Lock()
//	defer l.mu.Unlock()
//
//	now := l.clock.Now()
//
//	rate := l.rate.Duration()
//	window := now.Truncate(rate)
//
//	if l.window != window {
//		l.overCapacity = false
//		l.window = window
//	}
//
//	ttw := rate - now.Sub(window)
//
//	if !l.initialGotten {
//		l.initialGotten = true
//
//		l.window = now.Add(l.rate.Duration())
//		return 0, true
//	}
//
//	window := now.Add(l.rate.Duration())
//	if window != l.window {
//		// we are in the next window
//		l.overCapacity = false
//	}
//
//	l.overCapacity = l.slotsUsed >= l.capacity
//
//	l.slotsUsed++
//	l.slotsAvailable--
//
//	now := l.clock.Now()
//
//	ttw = time.Duration(0)
//	pass = true
//
//	if l.deadline.Before(now) {
//		// Deadline to refill reached
//		l.slotsAvailable = l.capacity
//	}
//
//	if l.overCapacity {
//		l.deadline = now.Add(l.rate.Duration())
//		ttw = l.rate.Duration() - now.Sub(l)
//		pass = false
//		return
//	}
//
//	pass = false
//	return
//}
//
//func NewFixedWindowRateLimiter(args FixedWindowArgs) FixedWindowRateLimiter {
//	return FixedWindowRateLimiter{
//		capacity:   args.Capacity,
//		rate:       args.Rate,
//		windowFunc: windowFuncFactory(args),
//	}
//}
//
