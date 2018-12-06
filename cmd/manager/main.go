package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/VirtusLab/jenkins-operator/pkg/apis"
	"github.com/VirtusLab/jenkins-operator/pkg/controller"
	"github.com/VirtusLab/jenkins-operator/pkg/log"
	"github.com/VirtusLab/jenkins-operator/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

func printInfo() {
	log.Log.Info(fmt.Sprintf("Version: %s", version.Version))
	log.Log.Info(fmt.Sprintf("Git commit: %s", version.GitCommit))
	log.Log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	minikube := flag.Bool("minikube", false, "Use minikube as a Kubernetes platform")
	local := flag.Bool("local", false, "Run operator locally")
	debug := flag.Bool("debug", false, "Set log level to debug")
	flag.Parse()

	log.SetupLogger(debug)
	printInfo()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Log.Error(err, "failed to get watch namespace")
		os.Exit(-1)
	}
	log.Log.Info(fmt.Sprintf("watch namespace: %v", namespace))

	sdk.ExposeMetricsPort()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Log.Error(err, "failed to get config")
		os.Exit(-1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Log.Error(err, "failed to create manager")
		os.Exit(-1)
	}

	log.Log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Log.Error(err, "failed to setup scheme")
		os.Exit(-1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, *local, *minikube); err != nil {
		log.Log.Error(err, "failed to setup controllers")
		os.Exit(-1)
	}

	log.Log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Log.Error(err, "failed to start cmd")
		os.Exit(-1)
	}
}
