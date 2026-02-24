package g

import (
	"fmt"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/signmem/go-log/log"
	"time"
)

var (
	Logger *log.Logger
	Ipaddr string
)

func InitLog() *log.Logger {
	LogMaxAge := Config().LogMaxAge
	LogRotateAge := Config().LogRotateAge
	logfile := Config().LogFile
	writer, err := rotatelogs.New(
		fmt.Sprintf("%s.%s", logfile, "%Y%m%d_%H%M%S.log"),
		rotatelogs.WithLinkName(logfile),
		rotatelogs.WithMaxAge(time.Second * time.Duration(LogMaxAge)),
		rotatelogs.WithRotationTime(time.Second * time.Duration(LogRotateAge)),
	)

	if err != nil {
		panic(fmt.Errorf("error opening file: %v", err))
	}

	Logger := log.NewSimple(
		log.WriterSink(writer,
			"[%s] [%s] [%d] [%s:%d] >>> [%s] msg=%s\n",
			[]string{"full_time", "priority", "pid", "filename", "lineno",
				"executable", "message"}))

	return Logger
}
