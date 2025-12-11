/*
Copyright 2025.

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	downloadv1 "github.com/yunuskaya-08/ytdlp-operator/api/v1"
)

const (
	timeout           = time.Second * 10
	interval          = time.Millisecond * 250
	resourceName      = "test-download-basic"
	resourceNamespace = "default"
)

var _ = Describe("Download Controller", func() {
	Context("When creating a basic Download resource (TDD RED Phase)", func() {
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		download := &downloadv1.Download{}

		jobName := resourceName + "-worker"
		jobLookupKey := types.NamespacedName{Name: jobName, Namespace: resourceNamespace}
		createdJob := &batchv1.Job{}

		BeforeEach(func() {
			By("Creating the custom resource with a VALID Spec")
			err := k8sClient.Get(ctx, typeNamespacedName, download)
			if err != nil && errors.IsNotFound(err) {
				resource := &downloadv1.Download{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: downloadv1.DownloadSpec{
						InputURL: "https://example.com/video",
						Output: downloadv1.OutputConfig{
							S3: &downloadv1.S3Output{
								Bucket:    "test-bucket",
								Key:       "video.mp4",
								SecretRef: "s3-creds",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &downloadv1.Download{ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: resourceNamespace}}
			By("Cleanup the specific resource instance Download")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			Eventually(func() bool {
				return errors.IsNotFound(k8sClient.Get(ctx, typeNamespacedName, &downloadv1.Download{}))
			}, timeout, interval).Should(BeTrue())
		})

		It("should successfully create and own a Kubernetes Job", func() {

			// Re-fetch the reconciler instance (important for integration testing setup)
			controllerReconciler := &DownloadReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// CRITICAL STEP: Manually trigger the reconciliation loop once.
			By("Manually calling Reconcile to trigger Job creation")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// Expect the Reconcile function to succeed (return no error)
			Expect(err).NotTo(HaveOccurred(), "Reconcile failed during initial Job creation attempt")

			// Now, we assert that the Job was created.
			By("Asserting the expected Job has been created")

			// The Job should now be found immediately.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, jobLookupKey, createdJob)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "The controller failed to create the expected Job within the timeout.")
		})
	})
})
