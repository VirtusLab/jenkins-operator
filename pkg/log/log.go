package log

// FIXME delete after bump to v0.2.0 version

import (
	"log"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Log represents global logger
var Log logr.Logger

const (
	// VWarn defines warning log level
	VWarn = -1
	// VDebug defines debug log level
	VDebug = 1
)

// SetupLogger setups global logger
func SetupLogger(development *bool) {
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
		log.Fatal(err)
	}

	Log = zapr.NewLogger(zapLog).WithName("jenkins-operator")
	// Enable logging in controller-runtime, without this you won't get logs when reconcile loop return an error
	runtimelog.SetLogger(Log)
}
