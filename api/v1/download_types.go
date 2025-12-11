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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DownloadSpec defines the desired state of Download
type DownloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// InputURL is the source URL for yt-dlp to download (e.g., a YouTube or Vimeo link).
	// +kubebuilder:validation:Required
	InputURL string `json:"inputURL"`

	// FormatSelection determines the video/audio quality and codec options.
	// This maps directly to yt-dlp's --format/-f flag syntax.
	// Example: "bestvideo[height<=1080]+bestaudio/best[height<=1080]"
	// +optional
	FormatSelection string `json:"formatSelection,omitempty"`

	// PostProcessing contains options for post-download operations like conversion.
	// +optional
	PostProcessing *PostProcessingConfig `json:"postProcessing,omitempty"`

	// OutputTarget specifies where the final file should be stored.
	// +kubebuilder:validation:Required
	Output OutputConfig `json:"output"`
}

// DownloadStatus defines the observed state of Download.
type DownloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase represents the current state of the download job
	// (e.g., Pending, Running, Completed, Failed).
	// This tracks the overall lifecycle of the Download resource.
	// +optional
	Phase string `json:"phase,omitempty"`

	// JobName is the name of the Kubernetes Job/Pod running the download.
	// This is useful for debugging and tracking the underlying resource.
	// +optional
	JobName string `json:"jobName,omitempty"`

	// DownloadRate reports the last known download speed (e.g., "10.3 MiB/s").
	// Maps to bytes_per_second in the pseudo-manifest.
	// +optional
	DownloadRate string `json:"downloadRate,omitempty"`

	// CompletionTime records when the download finished.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// ErrorMessage is set if the Phase is 'Failed'.
	// +optional
	ErrorMessage string `json:"errorMessage,omitempty"`

	// conditions represent the current state of the Download resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PostProcessingConfig defines steps like audio extraction or embedding metadata.
type PostProcessingConfig struct {
	// ExtractAudio converts the file to audio-only (e.g., mp3 or opus).
	// Corresponds to yt-dlp '--extract-audio'.
	// +optional
	ExtractAudio bool `json:"extractAudio,omitempty"`

	// AudioFormat specifies the output format for audio extraction (e.g., "mp3", "flac").
	// Corresponds to yt-dlp '--audio-format'.
	// +optional
	AudioFormat string `json:"audioFormat,omitempty"`
}

// OutputConfig defines the destination for the downloaded file.
type OutputConfig struct {
	// S3 specifies the S3 bucket destination.
	// +optional
	S3 *S3Output `json:"s3,omitempty"`

	// TODO: Add other outputs like PVC or Azure Blob here later
}

type S3Output struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// Key is the full path/filename within the bucket (e.g., "videos/myvideo.mp4").
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// SecretRef refers to a Kubernetes Secret containing credentials for the S3 target.
	// The Secret must contain keys like 'accessKeyID' and 'secretAccessKey'.
	// +kubebuilder:validation:Required
	SecretRef string `json:"secretRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Download is the Schema for the downloads API
type Download struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Download
	// +required
	Spec DownloadSpec `json:"spec"`

	// status defines the observed state of Download
	// +optional
	Status DownloadStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DownloadList contains a list of Download
type DownloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Download `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Download{}, &DownloadList{})
}
