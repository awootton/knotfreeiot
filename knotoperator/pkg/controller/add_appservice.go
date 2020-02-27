package controller

import "github.com/awootton/knotfreeiot/knotoperator/pkg/controller/appservice"

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, appservice.Add)
}
