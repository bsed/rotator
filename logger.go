package elog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"strconv"

	"github.com/mattn/go-isatty"
	"github.com/valyala/fasttemplate"

	"github.com/labstack/gommon/color"
	"github.com/labstack/gommon/log"
)

type (
	Logger struct {
		prefix     string
		level      log.Lvl
		output     io.Writer
		template   *fasttemplate.Template
		levels     []string
		color      *color.Color
		bufferPool sync.Pool
		mutex      sync.Mutex
		rotator    Rotator
	}
)

const (
	DEBUG log.Lvl = iota + 1
	INFO
	WARN
	ERROR
	OFF
)

var (
	defaultHeader = `{"time":"${time_rfc3339}","level":"${level}","prefix":"${prefix}",` +
		`"file":"${short_file}","line":"${line}"}`
)

func NewLogger(path, prefix string, limitSize int) (l *Logger) {
	rotator := NewFileSizeRotator(path, prefix, "log", limitSize)
	l = &Logger{
		level:    INFO,
		prefix:   prefix,
		template: l.newTemplate(defaultHeader),
		color:    color.New(),
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 256))
			},
		},
		rotator: rotator,
	}
	l.initLevels()
	writer, err := rotator.GetNextWriter()
	if err != nil {
		panic(err)
	}
	l.SetOutput(writer)
	return
}

func (l *Logger) initLevels() {
	l.levels = []string{
		"-",
		l.color.Blue("DEBUG"),
		l.color.Green("INFO"),
		l.color.Yellow("WARN"),
		l.color.Red("ERROR"),
	}
}

func (l *Logger) newTemplate(format string) *fasttemplate.Template {
	return fasttemplate.New(format, "${", "}")
}

func (l *Logger) DisableColor() {
	l.color.Disable()
	l.initLevels()
}

func (l *Logger) EnableColor() {
	l.color.Enable()
	l.initLevels()
}

func (l *Logger) Prefix() string {
	return l.prefix
}

func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

func (l *Logger) Level() log.Lvl {
	return l.level
}

func (l *Logger) SetLevel(v log.Lvl) {
	l.level = v
}

func (l *Logger) Output() io.Writer {
	return l.output
}

func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
	if w, ok := w.(*os.File); !ok || !isatty.IsTerminal(w.Fd()) {
		l.DisableColor()
	}
}

func (l *Logger) Color() *color.Color {
	return l.color
}

func (l *Logger) SetHeader(h string) {
	l.template = l.newTemplate(h)
}

func (l *Logger) Print(i ...interface{}) {
	l.log(0, "", i...)
	// fmt.Fprintln(l.output, i...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.log(0, format, args...)
}

func (l *Logger) Printj(j log.JSON) {
	l.log(0, "json", j)
}

func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Debugj(j log.JSON) {
	l.log(DEBUG, "json", j)
}

func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Infoj(j log.JSON) {
	l.log(INFO, "json", j)
}

func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Warnj(j log.JSON) {
	l.log(WARN, "json", j)
}

func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Errorj(j log.JSON) {
	l.log(ERROR, "json", j)
}

func (l *Logger) Fatal(i ...interface{}) {
	l.Print(i...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Printf(format, args...)
	os.Exit(1)
}

func (l *Logger) Fatalj(j log.JSON) {
	l.Printj(j)
	os.Exit(1)
}

func (l *Logger) Panic(i ...interface{}) {
	l.Print(i...)
	panic(fmt.Sprint(i...))
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Printf(format, args...)
	panic(fmt.Sprintf(format, args))
}

func (l *Logger) Panicj(j log.JSON) {
	l.Printj(j)
	panic(j)
}

func (l *Logger) log(v log.Lvl, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	buf := l.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer l.bufferPool.Put(buf)
	_, file, line, _ := runtime.Caller(3)

	if v >= l.level || v == 0 {
		message := ""
		if format == "" {
			message = fmt.Sprint(args...)
		} else if format == "json" {
			b, err := json.Marshal(args[0])
			if err != nil {
				panic(err)
			}
			message = string(b)
		} else {
			message = fmt.Sprintf(format, args...)
		}

		_, err := l.template.ExecuteFunc(buf, func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "time_rfc3339":
				return w.Write([]byte(time.Now().Format(time.RFC3339)))
			case "level":
				return w.Write([]byte(l.levels[v]))
			case "prefix":
				return w.Write([]byte(l.prefix))
			case "long_file":
				return w.Write([]byte(file))
			case "short_file":
				return w.Write([]byte(path.Base(file)))
			case "line":
				return w.Write([]byte(strconv.Itoa(line)))
			}
			return 0, nil
		})

		if err == nil {
			s := buf.String()
			i := buf.Len() - 1
			if s[i] == '}' {
				// JSON header
				buf.Truncate(i)
				buf.WriteByte(',')
				if format == "json" {
					buf.WriteString(message[1:])
				} else {
					buf.WriteString(`"message":`)
					buf.WriteString(strconv.Quote(message))
					buf.WriteString(`}`)
				}
			} else {
				// Text header
				buf.WriteByte(' ')
				buf.WriteString(message)
			}
			buf.WriteByte('\n')
			n, err := l.output.Write(buf.Bytes())
			if err == nil && l.rotator.ReachLimit(n) {
				w, err := l.rotator.GetNextWriter()
				if err != nil {
					l.Error(err)
				} else {
					l.SetOutput(w)
				}
			}
		}
	}
}
