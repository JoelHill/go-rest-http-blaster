package cbapiclient

/*
This file can be removed when the logrus dependency is removed
*/
import (
	"context"

	"github.com/InVisionApp/go-logger"
	"github.com/InVisionApp/go-logger/shims/logrus"
	logrusOrig "github.com/sirupsen/logrus"
)

// convert the logrus entry in context
func logrusShim(ctx context.Context) log.Logger {
	logrusEntry, ok := pkgCtxLoggerProviderFunc(ctx)
	if !ok {
		logrusEntry = logrusOrig.NewEntry(logrusOrig.New())
	}

	return logrus.New(logrusEntry.Logger)
}
