package internal

func AtLeast(n uint64) func(uint64) uint64 {
	return func(m uint64) uint64 {
		if m < n {
			return n
		}
		return m
	}
}
