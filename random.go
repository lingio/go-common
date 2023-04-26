package common

// Shuffle calls swap len-1 times to swap index i with j.
func Shuffle(
	rng interface{ Intn(int) int },
	len int,
	swap func(i, j int),
) {
	// This shuffle is known as Fisher-Yates.
	for i := len - 1; i > 0; i-- {
		j := rng.Intn(len)
		swap(i, j)
	}
}
