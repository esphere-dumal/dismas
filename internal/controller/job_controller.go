package controller

import (
	"context"
	"errors"
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

// Event describes a Job.
type Event struct {
	Command string
	Args    []string
}

// IsEqual tell wether two Event is same (Command and args).
func (lhs Event) isEqual(rhs Event) bool {
	if lhs.Command != rhs.Command {
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

// JobReconciler reconciles a Job object.
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	LastEvents map[string]map[string]Event
	Podname    string
}

//+kubebuilder:rbac:groups=crd.dismas.org,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.dismas.org,resources=jobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.dismas.org,resources=jobs/finalizers,verbs=update

const maxRetry = 7

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	logr.Info("Receiving an event to handle for job " + req.Name)

	// 1. Fetch the job and deal with deleted one
	var job dismasv1.Job
	err := r.Get(ctx, req.NamespacedName, &job)
	if apierrors.IsNotFound(err) {
		r.cleanJobCache(req.NamespacedName)
		logr.Info("Remove cache for deleted object " + req.Name)

		return reconcileDone(), nil
	} else if err != nil {
		return requeueAfter(), client.IgnoreNotFound(err)
	}

	// 2. Check if the job is repeat
	if r.isRepeatJob(Event{Command: job.Spec.Command, Args: job.Spec.Args}, req.NamespacedName) {
		logr.Info("Refuse to exec a repeat job " + req.Name)
		return reconcileDone(), nil
	}

	// 3. Execute the command
	logr.Info("Execute command for " + req.Name)

	stdout, stderr, cmderr := r.execute(job.Spec.Command, job.Spec.Args)

	// 4. CAS update the status
	for retryTime := 0; retryTime <= maxRetry; retryTime++ {
		err := r.updateJobStatus(ctx, &job, stdout, stderr, cmderr)

		// Successfully updated status
		if err == nil {
			logr.Info("Updated job status for " + req.Name)
			return ctrl.Result{}, nil
		}

		// Unexpected errors
		if !apierrors.IsConflict(err) {
			logr.Error(err, "Updating occurred a non-conflict error for job "+req.Name)

			return ctrl.Result{}, err
		}

		// Conflict, retry updating
		logr.Info("Updating but occurred conflict, going to retry for " + req.Name)

		if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
			logr.Error(err, "Unable to re-fetch object")

			return requeueAfter(), client.IgnoreNotFound(err)
		}
	}

	return requeueAfter(), errors.New("too many conflicts when retring updating")
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dismasv1.Job{}).
		Complete(r)
}

// cleanJobCache removes a deleted Job's last event tracked in cache.
func (r *JobReconciler) cleanJobCache(namespacedname types.NamespacedName) {
	if _, ok := r.LastEvents[namespacedname.Namespace]; !ok {
		return
	}

	delete(r.LastEvents[namespacedname.Namespace], namespacedname.Name)
}

/*
isRepeatJob checks a job is the same one as last executed
under two conditions the cache will be updated and return false
a new job created or the command are different.
*/
func (r *JobReconciler) isRepeatJob(newEvent Event, namespacedname types.NamespacedName) bool {
	isNamespaceExist := func() bool {
		if _, ok := r.LastEvents[namespacedname.Namespace]; ok {
			return true
		}

		return false
	}

	// There is no target namespace in cache -> a create event
	if !isNamespaceExist() {
		r.LastEvents[namespacedname.Namespace] = make(map[string]Event)
		r.LastEvents[namespacedname.Namespace][namespacedname.Name] = newEvent

		return false
	}

	lastEvent, ok := r.LastEvents[namespacedname.Namespace][namespacedname.Name]
	// There is no target name in cache or There is different between two events
	if !ok || !lastEvent.isEqual(newEvent) {
		r.LastEvents[namespacedname.Namespace][namespacedname.Name] = newEvent

		return false
	}

	return true
}

// execute run a command with its args.
func (r *JobReconciler) execute(command string, args []string) (string, string, error) {
	cmd := exec.Command(command, args...)

	var output, cmderr strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &cmderr

	err := cmd.Run()

	return output.String(), cmderr.String(), err
}

// checkJobMapIsNil make sures maps in Job is initialized.
func checkJobMapIsNotNil(job *dismasv1.Job) {
	if job.Status.Stdouts == nil {
		job.Status.Stdouts = make(map[string]string)
	}

	if job.Status.Stderrs == nil {
		job.Status.Stderrs = make(map[string]string)
	}

	if job.Status.Errors == nil {
		job.Status.Errors = make(map[string]string)
	}
}

// updateJobStatus checks job and updates datas.
func (r *JobReconciler) updateJobStatus(ctx context.Context, job *dismasv1.Job,
	stdout string, stderr string, err error) error {
	checkJobMapIsNotNil(job)

	job.Status.Stdouts[r.Podname] = stdout
	job.Status.Stderrs[r.Podname] = stderr

	errorMessage := "No error when executing the command + " + job.Name
	if err != nil {
		errorMessage = err.Error()
	}

	job.Status.Errors[r.Podname] = errorMessage

	log.Log.Info("Job updated in cache, going to update CR for " + job.Name)

	return r.Status().Update(ctx, job)
}
