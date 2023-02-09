/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"os/exec"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dismasv1 "dismas/api/v1"
)

// Event describes a Job
type Event struct {
	Command string
	Args    []string
}

// IsEqual tell wether two Event is same (Command and args)
func (lhs Event) IsEqual(rhs Event) bool {
	if lhs.Command != lhs.Command {
		return false
	}

	if len(lhs.Args) != len(rhs.Args) {
		return false
	}

	for idx, arg := range lhs.Args {
		if arg != rhs.Args[idx] {
			return false
		}
	}

	return true
}

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	LastEvents map[string]map[string]Event
	Podname    string
}

//+kubebuilder:rbac:groups=dismas.dismas.esphe,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dismas.dismas.esphe,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dismas.dismas.esphe,resources=jobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.Info("Receiving an event to handle")

	// 1. Fetch the job
	var job dismasv1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		lg.Error(err, "Unable to fetch object")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Check if the job is repeat
	// TODO: How to tell from a repeat job or a deleted-new-created job
	if r.isRepeatJob(Event{Command: job.Spec.Command, Args: job.Spec.Args}, req.NamespacedName) {
		lg.Info("Refuse to exec a repeat job")
		return ctrl.Result{}, nil
	}

	// 3. Execute the command
	stdout, stderr, cmderr := r.execute(job.Spec.Command, job.Spec.Args)
	lg.Info("Executed command for " + req.Name)
	if cmderr != nil {
		lg.Error(cmderr, cmderr.Error())
	}

	// 4. CAS update the status
	checkJobMapIsNotNil(&job)
	for {
		if err := r.updateJobStatus(&job, stdout, stderr, cmderr, ctx); err == nil {
			// Successfully updated
			lg.Info("Updated job status for " + req.Name)

			return ctrl.Result{}, nil

		} else if apierrors.IsConflict(err) {
			// Conflict, re-get object and retry updating
			lg.Info("Updating but occured confilct, going to retry for " + req.Name)

			if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
				lg.Error(err, "Unable to re-fetch object")
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
		} else {
			// Unexpected error
			lg.Error(err, "Updating occured a non-conflict error")

			return ctrl.Result{}, err
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dismasv1.Job{}).
		Complete(r)
}

/*
isRepeatJob checks a job is the same one as last executed
- a new job created: track in cache
- a job running but the CR is updated: update records in memory
TODO: a job finished running
*/
func (r *JobReconciler) isRepeatJob(newEvent Event, namespacedname types.NamespacedName) bool {
	// There is no target namespace in cache
	if _, ok := r.LastEvents[namespacedname.Namespace]; !ok {
		r.LastEvents[namespacedname.Namespace] = make(map[string]Event)
		r.LastEvents[namespacedname.Namespace][namespacedname.Name] = newEvent
		return false
	}

	lastEvent, ok := r.LastEvents[namespacedname.Namespace][namespacedname.Name]
	// There is no target name in cache or There is different between two events
	if !ok || !lastEvent.IsEqual(newEvent) {
		r.LastEvents[namespacedname.Namespace][namespacedname.Name] = newEvent
		return false
	}

	return true
}

// execute run a command with its args
func (r *JobReconciler) execute(command string, args []string) (string, string, error) {
	cmd := exec.Command(command, args...)

	var output, cmderr strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &cmderr

	err := cmd.Run()
	return output.String(), cmderr.String(), err
}

// checkJobMapIsNil make sures maps in a Job object is initialized
func checkJobMapIsNotNil(job *dismasv1.Job) {
	if job.Status.Stdouts == nil {
		job.Status.Stdouts = make(map[string]string)
	}

	if job.Status.Stderrs == nil {
		job.Status.Stdouts = make(map[string]string)
	}

	if job.Status.Errors == nil {
		job.Status.Errors = make(map[string]string)
	}
}

// updateJobStatus checks job and updates datas
func (r *JobReconciler) updateJobStatus(job *dismasv1.Job, stdout string, stderr string, err error, ctx context.Context) error {
	job.Status.Stdouts[r.Podname] = stdout
	job.Status.Stderrs[r.Podname] = stderr
	job.Status.Errors[r.Podname] = err.Error()

	return r.Status().Update(ctx, job)
}
