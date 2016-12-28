package elog

import "testing"

func TestLogger(t *testing.T) {
	l := NewLogger("", "app", 1<<10)

	for i := 0; i < 15; i++ {
		l.Infof("ttt %d", i)
	}
}
