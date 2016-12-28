package elog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	defaultSize   = 100 << 20 // 100MB
	defaultFormat = "2006-01-02_150405"
)

type FileSizeRotator struct {
	path string
	// file name prefix
	prefixName string
	extName    string
	format     string
	// size
	currSize  uint64
	limitSize uint64

	fd io.WriteCloser
}

func NewFileSizeRotator(path, prefix, ext string, limitSize int) *FileSizeRotator {
	if prefix == "" {
		prefix = "app"
	}
	if ext == "" {
		ext = "log"
	}
	if limitSize == 0 {
		limitSize = defaultSize
	}
	return &FileSizeRotator{
		path:       path,
		prefixName: prefix,
		extName:    ext,
		format:     defaultFormat,
		limitSize:  uint64(limitSize),
	}
}

// ReachLimit checks if current size is bigger than limit size
func (r *FileSizeRotator) ReachLimit(n int) bool {
	atomic.AddUint64(&r.currSize, uint64(n))
	if r.currSize > r.limitSize {
		return true
	}
	return false
}

func (r *FileSizeRotator) getNextName() string {
	t := time.Now()
	timeStr := t.Format(r.format)
	file := fmt.Sprintf("%s_%s_%d.%s", r.prefixName, timeStr, r.currSize, r.extName)
	return filepath.Join(r.path, file)
}

func (r *FileSizeRotator) GetNextWriter() (io.Writer, error) {
	file := r.getNextName()

	perm, err := strconv.ParseInt("0755", 8, 64)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err == nil {
		// Make sure file perm is user set perm cause of `os.OpenFile` will obey umask
		os.Chmod(file, os.FileMode(perm))

		// close old fd
		if r.fd != nil {
			r.fd.Close()
		}
		r.fd = fd

		// reset currSize
		r.currSize = 0
	} else {
		return nil, err
	}

	return fd, nil
}

func (r *FileSizeRotator) GetCurrentCloser() io.WriteCloser {
	return r.fd
}
