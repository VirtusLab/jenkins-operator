// TODO delete after resolve issue https://github.com/operator-framework/operator-sdk/issues/503
package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var Log logr.Logger

// ZapLogger is a Logger implementation.
// If development is true, a Zap development config will be used
// (stacktraces on warnings, no sampling), otherwise a Zap production
// config will be used (stacktraces on errors, sampling).
func SetupLogger(development *bool) error {
	var zapLog *zap.Logger
	var err error

	if *development {
		zapLogCfg := zap.NewDevelopmentConfig()
		zapLog, err = zapLogCfg.Build(zap.AddCallerSkip(1))
	} else {
		zapLogCfg := zap.NewProductionConfig()
		zapLog, err = zapLogCfg.Build(zap.AddCallerSkip(1))
	}
	if err != nil {
		return err
	}

	Log = zapr.NewLogger(zapLog)
	return nil
}
