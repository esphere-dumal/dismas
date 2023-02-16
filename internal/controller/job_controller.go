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

	// TODO: using a map[string]Event to replace map[string]map[string]Event
	// TODO: map key
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
	// TODO: only in debug
	logr.Info("Receiving an event to handle for job " + req.Name)

	// 1. Fetch the job and deal with delete event
	var job dismasv1.Job
	err := r.Get(ctx, req.NamespacedName, &job)
	if apierrors.IsNotFound(err) {
		r.processDelete(ctx, req.Namespace, req.Name)
		logr.Info("already removed job: " + req.Name)

		return reconcileDone(), nil
	} else if err != nil {
		return requeueAfter(), client.IgnoreNotFound(err)
	}

	// 2. Deal with update event triggered by operator itselef
	if r.isRepeatJob(Event{Command: job.Spec.Command, Args: job.Spec.Args}, req.NamespacedName) {
		logr.Info("Refuse to exec a repeat job " + req.Name)
		return reconcileDone(), nil
	}

	// 3. Deal with create or update event, execute the command
	logr.Info("Execute command for " + req.Name)

	stdout, stderr, cmderr := r.execute(job.Spec.Command, job.Spec.Args)

	// 4. CAS update the status
	// TODO: use a function here
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
		// TODO: warning or others, log level
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

// processDelete removes a deleted Job's last event tracked in cache.
func (r *JobReconciler) processDelete(ctx context.Context, namespace string, name string) {
	log.Log.Info("delete " + name)

	if _, ok := r.LastEvents[namespace]; ok {
		delete(r.LastEvents[namespace], name)
		log.Log.Info("delete cache for " + name)
	}
}

// isRepeatJob checks a job is the same one as last executed
// TODO: better func behaviour
func (r *JobReconciler) isRepeatJob(newEvent Event, namespacedname types.NamespacedName) bool {
	// TODO: better judgement

	// ensure not a nil map
	// removed an exit
	namespaceMap, ok := r.LastEvents[namespacedname.Namespace]
	if !ok {
		namespaceMap = make(map[string]Event)
		namespaceMap[namespacedname.Name] = newEvent

		r.LastEvents[namespacedname.Name] = namespaceMap
	}

	// There is no target name in cache or There is different between two events
	lastEvent, ok := namespaceMap[namespacedname.Name]
	if !ok || !lastEvent.isEqual(newEvent) {
		namespaceMap[namespacedname.Name] = newEvent

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
