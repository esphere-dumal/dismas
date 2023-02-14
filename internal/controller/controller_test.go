/*

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
// +kubebuilder:docs-gen:collapse=Apache License

package controller

import (
	"context"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dismasv1 "dismas/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("CronJob controller", func() {
	var ctx = context.TODO()

	Context("When create Job", func() {
		var (
			managerCtx  context.Context
			managerStop func()
			wg          sync.WaitGroup
		)

		BeforeEach(func() {
			managerCtx, managerStop = context.WithCancel(context.Background())

			k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:             scheme.Scheme,
				MetricsBindAddress: "0",
			})
			Expect(err).NotTo(HaveOccurred())

			err = (&JobReconciler{
				Podname: Podname,
			}).SetupWithManager(k8sManager)
			Expect(err).NotTo(HaveOccurred())

			wg = sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer GinkgoRecover()
				err := k8sManager.Start(managerCtx)
				Expect(err).ToNot(HaveOccurred(), "failed to run manager")
				wg.Done()
			}()
		})

		AfterEach(func() {
			Eventually(func() error {
				err := k8sClient.DeleteAllOf(ctx, &dismasv1.Job{}, client.InNamespace(testNamespace))
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}

				list := &dismasv1.JobList{}

				err = k8sClient.List(ctx, list, client.InNamespace(testNamespace))
				if err != nil {
					return err
				}

				if len(list.Items) != 0 {
					return errors.Errorf("expected BackupExecutionList to be empty, but got length %d", len(list.Items))
				}

				return nil
			}, timeout, interval).ShouldNot(HaveOccurred())

			managerStop()
			wg.Wait()
		})

		It("Should create a new job", func() {
			const (
				jobName = "test-echo"
				command = "echo test"
			)

			By("Creating an echo job")
			job := &dismasv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      jobName,
				},
				Spec: dismasv1.JobSpec{
					Command: command,
				},
			}
			Expect(k8sClient.Create(ctx, job)).Should(Succeed())

			By("Check job output")
			AssertJobStatus(ctx, *job, "test")
		})

	})
})

func AssertJobStatus(ctx context.Context, job dismasv1.Job, output string) {
	By(fmt.Sprintf("By checking job %s output to be %s", job.Name, output))

	Eventually(func() error {
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(&job), &job)
		if err != nil {
			return err
		}

		if _, ok := job.Status.Stdouts[Podname]; !ok {
			return errors.New("Not updated yet")
		}

		if job.Status.Stdouts[Podname] != output {
			return errors.New("Not target output")
		}

		return nil
	}, timeout, interval).ShouldNot(HaveOccurred())
}
