package controller

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dismasv1 "dismas/api/v1"
)

var _ = Describe("Job Controller", func() {

	// Define objects name and spec
	const (
		JobName      = "testjob"
		JobNamespace = "default"

		command = "ls"
	)

	Context("When updating Job Status", func() {
		It("Should updated Job Status about Stdouts", func() {
			By("By creating a new job with a controller")
			ctx := context.Background()
			job := &dismasv1.Job{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "dismas.dismas.esphe/v1",
					Kind:       "Job",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: JobNamespace,
				},
				Spec: dismasv1.JobSpec{
					Command: command,
				},
			}

			createdJob := &dismasv1.Job{}
			kind := reflect.TypeOf(dismasv1.Job{}).Name()
			gvk := dismasv1.GroupVersion.WithKind(kind)
			controllerRef := metav1.NewControllerRef(createdJob, gvk)
			job.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, job)).Should(Succeed())

			jobLookupKey := types.NamespacedName{Name: JobName, Namespace: JobNamespace}

			By("Checking controller working")
			Expect(func() bool {
				var newJob dismasv1.Job
				err := k8sClient.Get(ctx, jobLookupKey, &newJob)
				if err != nil || newJob.Status.Stdouts == nil {
					return false
				}

				return true
			}).Should(BeTrue())
		})
	})
})
