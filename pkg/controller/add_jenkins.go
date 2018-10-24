package controller

import (
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, jenkins.Add)
}
