package log

import (
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Log represents global logger
var Log = log.Log.WithName("controller-jenkins")

const (
	// VWarn defines warning log level
	VWarn = -1
	// VDebug defines debug log level
	VDebug = 1
)

// SetupLogger setups global logger
func SetupLogger(development *bool) {
	logf.SetLogger(logf.ZapLogger(*development))
	Log = log.Log.WithName("controller-jenkins")
}
