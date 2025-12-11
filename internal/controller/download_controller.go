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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	downloadv1 "github.com/yunuskaya-08/ytdlp-operator/api/v1"
)

// Helper function to build the command-line arguments for yt-dlp
func buildYtDlpArgs(spec downloadv1.DownloadSpec) []string {
	args := []string{"yt-dlp", "--no-mtime", "--ignore-errors", "-o", "/data/%(title)s.%(ext)s"}

	// 1. Format Selection
	if spec.FormatSelection != "" {
		args = append(args, "--format", spec.FormatSelection)
	}

	// 2. Post-Processing
	if spec.PostProcessing != nil {
		if spec.PostProcessing.ExtractAudio {
			args = append(args, "--extract-audio")
			if spec.PostProcessing.AudioFormat != "" {
				args = append(args, "--audio-format", spec.PostProcessing.AudioFormat)
			}
		}
	}

	// 3. S3 Output Configuration
	if spec.Output.S3 != nil {
		args = append(args,
			"--upload-to", "s3-generic",
			"--s3-bucket", spec.Output.S3.Bucket,
		)
	}

	// 4. Input URL (always last)
	args = append(args, spec.InputURL)
	return args
}

// newDownloadJob creates the Kubernetes Job object for the download worker.
func (r *DownloadReconciler) newDownloadJob(download *downloadv1.Download, jobName string) (*batchv1.Job, error) {
	args := buildYtDlpArgs(download.Spec)

	// Environment variables for S3 credentials
	s3EnvVars := []corev1.EnvVar{
		{
			Name: "AWS_ACCESS_KEY_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: download.Spec.Output.S3.SecretRef},
					Key:                  "accessKeyID",
				},
			},
		},
		{
			Name: "AWS_SECRET_ACCESS_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: download.Spec.Output.S3.SecretRef},
					Key:                  "secretAccessKey",
				},
			},
		},
	}

	// Define a termination grace period to ensure yt-dlp finishes S3 upload on time
	var terminationGracePeriodSeconds int64 = 60

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: download.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:                 corev1.RestartPolicyOnFailure,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:  "yt-dlp-worker",
							Image: "yt-dlp/yt-dlp:latest",
							Args:  args,
							Env:   s3EnvVars,
						},
					},
				},
			},
		},
	}
	return job, nil
}

// DownloadReconciler reconciles a Download object
type DownloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=download.beebs.dev,resources=downloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=download.beebs.dev,resources=downloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=download.beebs.dev,resources=downloads/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Download object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *DownloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("Download", req.NamespacedName)

	// 1. Fetch the Download instance
	download := &downloadv1.Download{}
	if err := r.Get(ctx, req.NamespacedName, download); err != nil {
		// Ignore not found errors—the resource was likely deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- TDD Target Check: Job Exists? ---

	jobName := download.Name + "-worker"
	foundJob := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: download.Namespace}, foundJob)

	if err == nil {
		// Job found: The reconciliation loop should now monitor its status.
		log.Info("Worker Job already exists", "Job", foundJob.Name)

		// TODO: (Next step) We will add monitoring logic here.
		return ctrl.Result{}, nil
	}

	if client.IgnoreNotFound(err) != nil {
		// Actual error fetching the job (not just 'Not Found')
		log.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Job not found: Create it.
	job, err := r.newDownloadJob(download, jobName)
	if err != nil {
		log.Error(err, "Failed to construct Job object")
		return ctrl.Result{}, err
	}

	// Set the OwnerReference (Crucial for garbage collection)
	if err := controllerutil.SetControllerReference(download, job, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference")
		return ctrl.Result{}, err
	}

	// Create the Job
	log.Info("Creating a new Worker Job", "Job.Name", job.Name)
	if err = r.Create(ctx, job); err != nil {
		log.Error(err, "Failed to create Job")
		return ctrl.Result{}, err
	}

	// Job created successfully, requeue immediately to update status/monitor job.
	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DownloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&downloadv1.Download{}). // Primary resource we manage
		Owns(&batchv1.Job{}).        // Tell the manager we create and own Jobs
		Named("download").
		Complete(r)
}
