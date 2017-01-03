package rotator

import "testing"
import "github.com/labstack/gommon/log"

func Test_FileSizeRotator(t *testing.T) {
	l := log.New("-")
	fileRotator := NewFileSizeRotator("", "app", "log", 2000)
	l.SetOutput(fileRotator)
	l.SetLevel(log.DEBUG)

	for i := 0; i < 30; i++ {
		l.Infof("tess %d", i)
	}
}
