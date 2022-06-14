package pacemaker

func AtLeast(n int64) func(int64) int64 {
	return func(m int64) int64 {
		if m < n {
			return n
		}
		return m
	}
}
