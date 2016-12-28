package elog

import (
	"io"
)

type Rotator interface {
	// after doing log, input the written bytes number to check if reaching the limit
	ReachLimit(int) bool
	GetNextWriter() (io.Writer, error)
	GetCurrentCloser() io.WriteCloser
}
