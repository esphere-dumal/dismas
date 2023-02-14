package controller

import (
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

func reconcileDone() ctrl.Result {
	return ctrl.Result{}
}

// requeueAfter creates a ctrl.Result that requeue after secs seconds
// Default value: 3s.
func requeueAfter(secs ...int) ctrl.Result {
	if len(secs) > 1 {
		panic(fmt.Errorf("invalid argument secs"))
	}

	s := 3
	if len(secs) == 1 {
		s = secs[0]
	}

	return ctrl.Result{RequeueAfter: time.Duration(s) * time.Second}
}
