package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "tailpost.elastic.co", Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&TailpostAgent{},
		&TailpostAgentList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Register registers all types in this package into the given scheme
func Register(scheme *runtime.Scheme) error {
	return addKnownTypes(scheme)
}

// TailpostAgentSpec defines the desired state of TailpostAgent
type TailpostAgentSpec struct {
	// Replicas is the number of agents to run
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Image is the TailPost agent image to use
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy defines the policy for pulling the image
	// +optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// ServiceAccount is the name of the ServiceAccount to use
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// LogSources defines the log sources to collect
	LogSources []LogSourceSpec `json:"logSources"`

	// ServerURL is the endpoint to send logs to
	ServerURL string `json:"serverURL"`

	// BatchSize is the number of log lines to batch before sending
	// +optional
	BatchSize *int32 `json:"batchSize,omitempty"`

	// FlushInterval is the maximum time to hold a batch before sending
	// +optional
	FlushInterval string `json:"flushInterval,omitempty"`

	// Resource requirements for the TailPost agent
	// +optional
	Resources ResourceRequirementsSpec `json:"resources,omitempty"`
}

// LogSourceSpec defines a log source to collect
type LogSourceSpec struct {
	// Type is the type of log source (file, container, pod, etc.)
	Type string `json:"type"`

	// Path is the path to the log file (for file type)
	// +optional
	Path string `json:"path,omitempty"`

	// ContainerName is the name of the container to collect logs from (for container type)
	// +optional
	ContainerName string `json:"containerName,omitempty"`

	// PodSelector is a label selector to match pods (for pod type)
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// NamespaceSelector is a label selector to match namespaces (for pod type)
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// ResourceRequirementsSpec defines resource requirements
type ResourceRequirementsSpec struct {
	// Limits describes the maximum amount of compute resources allowed
	// +optional
	Limits ResourceListSpec `json:"limits,omitempty"`

	// Requests describes the minimum amount of compute resources required
	// +optional
	Requests ResourceListSpec `json:"requests,omitempty"`
}

// ResourceListSpec is a set of resource limits
type ResourceListSpec struct {
	// CPU is the CPU limit
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory is the memory limit
	// +optional
	Memory string `json:"memory,omitempty"`
}

// TailpostAgentStatus defines the observed state of TailpostAgent
type TailpostAgentStatus struct {
	// Conditions represent the latest available observations of the agent state
	// +optional
	Conditions []TailpostAgentCondition `json:"conditions,omitempty"`

	// AvailableReplicas represents the number of available agent replicas
	// +optional
	AvailableReplicas int32 `json:"availableReplicas"`

	// LastUpdateTime is the timestamp of the last status update
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// TailpostAgentCondition describes the state of a TailpostAgent at a certain point
type TailpostAgentCondition struct {
	// Type of condition
	Type string `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status string `json:"status"`

	// LastTransitionTime is the last time the condition transitioned from one status to another
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a unique, one-word, CamelCase reason for the condition's last transition
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable message indicating details about the transition
	// +optional
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TailpostAgent is the Schema for the tailpostagents API
type TailpostAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TailpostAgentSpec   `json:"spec,omitempty"`
	Status TailpostAgentStatus `json:"status,omitempty"`
}

// DeepCopyObject implements the runtime.Object interface
func (in *TailpostAgent) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy creates a deep copy of TailpostAgent
func (in *TailpostAgent) DeepCopy() *TailpostAgent {
	if in == nil {
		return nil
	}
	out := new(TailpostAgent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies TailpostAgent into the given object
func (in *TailpostAgent) DeepCopyInto(out *TailpostAgent) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TailpostAgentList contains a list of TailpostAgent
type TailpostAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TailpostAgent `json:"items"`
}

// DeepCopyObject implements the runtime.Object interface
func (in *TailpostAgentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy creates a deep copy of TailpostAgentList
func (in *TailpostAgentList) DeepCopy() *TailpostAgentList {
	if in == nil {
		return nil
	}
	out := new(TailpostAgentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies TailpostAgentList into the given object
func (in *TailpostAgentList) DeepCopyInto(out *TailpostAgentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TailpostAgent, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for TailpostAgentSpec
func (in *TailpostAgentSpec) DeepCopyInto(out *TailpostAgentSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.BatchSize != nil {
		in, out := &in.BatchSize, &out.BatchSize
		*out = new(int32)
		**out = **in
	}
	if in.LogSources != nil {
		in, out := &in.LogSources, &out.LogSources
		*out = make([]LogSourceSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopyInto for LogSourceSpec
func (in *LogSourceSpec) DeepCopyInto(out *LogSourceSpec) {
	*out = *in
	if in.PodSelector != nil {
		in, out := &in.PodSelector, &out.PodSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.NamespaceSelector != nil {
		in, out := &in.NamespaceSelector, &out.NamespaceSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto for ResourceRequirementsSpec
func (in *ResourceRequirementsSpec) DeepCopyInto(out *ResourceRequirementsSpec) {
	*out = *in
	in.Limits.DeepCopyInto(&out.Limits)
	in.Requests.DeepCopyInto(&out.Requests)
}

// DeepCopyInto for ResourceListSpec
func (in *ResourceListSpec) DeepCopyInto(out *ResourceListSpec) {
	*out = *in
}

// DeepCopyInto for TailpostAgentStatus
func (in *TailpostAgentStatus) DeepCopyInto(out *TailpostAgentStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]TailpostAgentCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyInto for TailpostAgentCondition
func (in *TailpostAgentCondition) DeepCopyInto(out *TailpostAgentCondition) {
	*out = *in
}
