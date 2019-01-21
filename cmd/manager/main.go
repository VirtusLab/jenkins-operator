package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/VirtusLab/jenkins-operator/pkg/apis"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins"
	"github.com/VirtusLab/jenkins-operator/pkg/event"
	"github.com/VirtusLab/jenkins-operator/pkg/log"
	"github.com/VirtusLab/jenkins-operator/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
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
		fatal(err, "failed to get watch namespace")
	}
	log.Log.Info(fmt.Sprintf("watch namespace: %v", namespace))

	// get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(err, "failed to get config")
	}

	// become the leader before proceeding
	err = leader.Become(context.TODO(), "jenkins-operator-lock")
	if err != nil {
		fatal(err, "failed to become leader")
	}

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		fatal(err, "failed to get ready.NewFileReady")
	}
	defer func() {
		_ = r.Unset()
	}()

	// create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		fatal(err, "failed to create manager")
	}

	log.Log.Info("Registering Components.")

	// setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		fatal(err, "failed to setup scheme")
	}

	// setup events
	events, err := event.New(cfg)
	if err != nil {
		fatal(err, "failed to create manager")
	}

	// setup Jenkins controller
	if err := jenkins.Add(mgr, *local, *minikube, events); err != nil {
		fatal(err, "failed to setup controllers")
	}

	log.Log.Info("Starting the Cmd.")

	// start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		fatal(err, "failed to start cmd")
	}
}

func fatal(err error, message string) {
	log.Log.Error(err, message)
	os.Exit(-1)
}
