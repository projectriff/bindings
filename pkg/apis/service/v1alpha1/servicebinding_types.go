package v1alpha1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/tracker"
	"knative.dev/pkg/webhook/psbinding"
)

const (
	ServiceBindingAnnotationKey = GroupName + "/service-binding"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBindingSpec   `json:"spec,omitempty"`
	Status ServiceBindingStatus `json:"status,omitempty"`
}

var (
	// Check that ServiceBinding can be validated and defaulted.
	_ apis.Validatable   = (*ServiceBinding)(nil)
	_ apis.Defaultable   = (*ServiceBinding)(nil)
	_ kmeta.OwnerRefable = (*ServiceBinding)(nil)

	// Check is Bindable
	_ psbinding.Bindable  = (*ServiceBinding)(nil)
	_ duck.BindableStatus = (*ServiceBindingStatus)(nil)
)

type ServiceBindingSpec struct {
	// Name of the service binding on disk, defaults to this resource's name
	Name string `json:"name,omitempty"`
	// Type of the provisioned service. The value is exposed directly as the
	// `type` in the mounted binding
	// +optional
	Type string `json:"type,omitempty`
	// Provider of the provisioned service. The value is exposed directly as the
	// `provider` in the mounted binding
	// +optional
	Provider string `json:"provider,omitempty`

	// Application resource to inject the binding into
	Application *ApplicationReference `json:"application,omitempty"`
	// Service referencing the binding secret
	Service *tracker.Reference `json:"service,omitempty"`

	// Env projects keys from the binding secret into the application as
	// environment variables
	Env []EnvVar `json:"env,omitempty"`

	// Mappings create new binding secret keys from literal values or templated
	// from existing keys
	Mappings []Mapping `json:"mappings,omitempty"`
}

type ApplicationReference struct {
	tracker.Reference

	// Containers to target within the application. If not set, all containers
	// will be injected. Containers may be specified by index or name.
	// InitContainers may only be specified by name.
	Containers []intstr.IntOrString `json:"containers,omitempty"`
}

type EnvVar struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Mapping struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ServiceBindingStatus struct {
	duckv1beta1.Status `json:",inline"`
	Binding            *corev1.LocalObjectReference `json:"binding,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

func (b *ServiceBinding) Validate(ctx context.Context) (errs *apis.FieldError) {
	errs = errs.Also(
		b.Spec.Application.Validate(ctx).ViaField("spec.application"),
	)
	if b.Spec.Application.Namespace != b.Namespace {
		errs = errs.Also(
			apis.ErrInvalidValue(b.Spec.Application.Namespace, "spec.application.namespace"),
		)
	}
	errs = errs.Also(
		b.Spec.Service.Validate(ctx).ViaField("spec.service"),
	)
	if b.Spec.Service.Namespace != b.Namespace {
		errs = errs.Also(
			apis.ErrInvalidValue(b.Spec.Service.Namespace, "spec.service.namespace"),
		)
	}
	if b.Spec.Service.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("spec.service.name"),
		)
	}
	for i, e := range b.Spec.Env {
		errs = errs.Also(
			e.Validate(ctx).ViaFieldIndex("env", i),
		)
		// TODO look for conflicting names
	}
	for i, m := range b.Spec.Mappings {
		errs = errs.Also(
			m.Validate(ctx).ViaFieldIndex("mappings", i),
		)
		// TODO look for conflicting names
	}

	return errs
}

func (e EnvVar) Validate(ctx context.Context) (errs *apis.FieldError) {
	if e.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("name"),
		)
	}
	if e.Key == "" {
		errs = errs.Also(
			apis.ErrMissingField("key"),
		)
	}

	return errs
}

func (m Mapping) Validate(ctx context.Context) (errs *apis.FieldError) {
	if m.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("name"),
		)
	}

	return errs
}

func (b *ServiceBinding) SetDefaults(context.Context) {
	if b.Spec.Name == "" {
		b.Spec.Name = b.Name
	}
	if b.Spec.Application.Namespace == "" {
		// Default the application's namespace to our namespace.
		b.Spec.Application.Namespace = b.Namespace
	}
	if b.Spec.Service.Namespace == "" {
		// Default the service's namespace to our namespace.
		b.Spec.Service.Namespace = b.Namespace
	}
}

func (b *ServiceBinding) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ServiceBinding")
}
