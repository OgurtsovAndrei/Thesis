package utils

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i, v := range ts {
		us[i] = f(v)
	}
	return us
}
