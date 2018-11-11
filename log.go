package buttonoff

import (
	"github.com/sirupsen/logrus"
)

var (
	appLogger = logrus.New()
)

func SetLogLevel(lvl logrus.Level) {
	appLogger.Level = lvl
}
