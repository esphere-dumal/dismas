package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dismasv1 "dismas/api/v1"
)

var _ = Describe("Job Controller", func() {

	// Define objects name and spec
	const (
		JobName      = "test-job"
		JobNamespace = "default"

		command = "ls"
	)

	Context("When updating Job Status", func() {
		It("Should updated Job Status about outputs", func() {

			By("By creating a new job")
			ctx := context.Background()
			job := &dismasv1.Job{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "dismas/v1",
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
			Expect(k8sClient.Create(ctx, job)).Should(Succeed())

		})
	})
})
