# Echo logger with rotatar

## Usage

```
package main

import ()
	"github.com/labstack/echo"
	elog "github.com/silentred/echo-log"
)

// Echo is the web engine
var Echo *echo.Echo

func init() {
	Echo = echo.New()
}

func main() {
	initLogger()
    Echo.Logger.Info("test")
	Echo.Start(":8090")
}

func initLogger() {
	path := "log"
	appName := "app"
	limitSize := 100 << 20 // 100MB

	Echo.Logger = elog.NewLogger(path, appName, limitSize)
	Echo.Logger.SetLevel(elog.WARN)

}

```