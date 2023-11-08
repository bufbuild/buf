package bufmodule

import "sync"

// onceThreeValues returns a function that invokes f only once and returns the values
// returned by f. The returned function may be called concurrently.
//
// If f panics, the returned function will panic with the same value on every call.
//
// This is copied from sync.OnceValues and extended to for three values.
func onceThreeValues[T1, T2, T3 any](f func() (T1, T2, T3)) func() (T1, T2, T3) {
	var (
		once  sync.Once
		valid bool
		p     any
		r1    T1
		r2    T2
		r3    T3
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		r1, r2, r3 = f()
		valid = true
	}
	return func() (T1, T2, T3) {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return r1, r2, r3
	}
}
