package zfasttrie

import (
	"fmt"
)

const debug = false

func First(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func FatalIf(err error) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf("FATAL: %v", err))
}

func Bug(format string, msg ...any) {
	if debug {
		panic(fmt.Sprintf(format, msg...))
	}
}

func BugOn(cond bool, format string, msg ...any) {
	if debug && cond {
		Bug(format, msg...)
	}
}

func BugOnNotEq(a, b any) {
	if a == b {
		return
	}
	Bug("BUG: a != b, %v != %v", a, b)
}
