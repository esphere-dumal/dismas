package controller

import (
	"context"
	"fmt"
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

	// LastEvents keeps track of namespace-name => lastevent
	LastEvents map[string]Event
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
	newEvent := Event{Command: job.Spec.Command, Args: job.Spec.Args}
	if !r.isRepeatEvent(newEvent, req.NamespacedName) {
		logr.Info("Refuse to exec a repeat job " + req.Name)
		return reconcileDone(), nil
	}

	// 3. Deal with create or update event, execute the command
	logr.Info("Execute command for " + req.Name)

	stdout, stderr, cmderr := r.execute(job.Spec.Command, job.Spec.Args)

	// 4. CAS update the status
	return r.tryUpdateStatus(ctx, job, stdout, stderr, cmderr, newEvent)
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

	key := NameToKey(namespace, name)
	if _, ok := r.LastEvents[key]; ok {
		delete(r.LastEvents, key)
		log.Log.Info("delete cache for ", key)
	}
}

// updateIfNotRepeatedEvent return false if is a repeated event
func (r *JobReconciler) isRepeatEvent(newEvent Event, namespacedname types.NamespacedName) bool {
	key := NameToKey(namespacedname.Namespace, namespacedname.Name)
	if lastEvent, ok := r.LastEvents[key]; ok {
		if lastEvent.isEqual(newEvent) {
			return false
		}
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

	if err != nil {
		job.Status.Errors[r.Podname] = err.Error()
	}

	return r.Status().Update(ctx, job)
}

func (r *JobReconciler) tryUpdateStatus(ctx context.Context, job dismasv1.Job,
	stdout string, stderr string, cmderr error, event Event) (ctrl.Result, error) {
	for retryTime := 0; retryTime <= maxRetry; retryTime++ {
		err := r.updateJobStatus(ctx, &job, stdout, stderr, cmderr)

		// Successfully updated status
		if err == nil {
			key := NameToKey(job.Namespace, job.Name)
			r.LastEvents[key] = event
			return reconcileDone(), nil
		}

		// Unexpected errors
		if !apierrors.IsConflict(err) {
			return reconcileDone(), err
		}

		// Conflict, retry updating
		if err := r.Get(ctx, types.NamespacedName{Namespace: job.Namespace, Name: job.Name}, &job); err != nil {
			return requeueAfter(), client.IgnoreNotFound(err)
		}
	}
	return reconcileDone(), nil
}

func NameToKey(namespace string, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
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
