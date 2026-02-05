package utils

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i, v := range ts {
		us[i] = f(v)
	}
	return us
}

func MapKeys[K comparable, V, W any](m map[K]V, f func(K) W) []W {
	ws := make([]W, len(m))
	i := 0
	for k := range m {
		ws[i] = f(k)
		i++
	}
	return ws
}

func MapValues[T comparable, U, V any](m map[T]U, f func(U) V) []V {
	vs := make([]V, len(m))
	i := 0
	for _, v := range m {
		vs[i] = f(v)
		i++
	}
	return vs
}

func MapEntries[T comparable, V, W any](m map[T]V, f func(T, V) W) []W {
	ws := make([]W, len(m))
	i := 0
	for k, v := range m {
		ws[i] = f(k, v)
		i++
	}
	return ws
}
