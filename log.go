package buttonoff

import (
	"github.com/Sirupsen/logrus"
)

var (
	appLogger = logrus.New()
)

func SetLogLevel(lvl logrus.Level) {
	appLogger.Level = lvl
}
