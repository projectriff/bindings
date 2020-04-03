package resolver

import (
	"context"
	"errors"
	"fmt"

	iduckv1alpha1 "github.com/projectriff/bindings/pkg/apis/duck/v1alpha1"
	"github.com/projectriff/bindings/pkg/client/injection/ducks/duck/v1alpha1/serviceable"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	pkgapisduck "knative.dev/pkg/apis/duck"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/tracker"
)

// URIResolver resolves Destinations and ObjectReferences into a URI.
type ServiceableResolver struct {
	tracker         tracker.Interface
	informerFactory pkgapisduck.InformerFactory
}

// NewServiceableResolver constructs a ServiceableResolver with context and a callback
// for a given ServiceableType passed to the ServiceableResolver's tracker.
func NewServiceableResolver(ctx context.Context, callback func(types.NamespacedName)) *ServiceableResolver {
	ret := &ServiceableResolver{}

	ret.tracker = tracker.New(callback, controller.GetTrackerLease(ctx))
	ret.informerFactory = &pkgapisduck.CachedInformerFactory{
		Delegate: &pkgapisduck.EnqueueInformerFactory{
			Delegate:     serviceable.Get(ctx),
			EventHandler: controller.HandleAll(ret.tracker.OnChanged),
		},
	}
	return ret
}

func (r *ServiceableResolver) ServiceableFromObjectReference(ref *tracker.Reference, parent interface{}) (*iduckv1alpha1.ServiceableBinding, error) {
	if ref == nil {
		return nil, errors.New("ref is nil")
	}
	if err := r.tracker.TrackReference(*ref, parent); err != nil {
		return nil, fmt.Errorf("failed to track %+v: %v", ref, err)
	}
	gvr, _ := meta.UnsafeGuessKindToResource(ref.GroupVersionKind())
	_, lister, err := r.informerFactory.Get(gvr)
	if err != nil {
		return nil, err
	}
	obj, err := lister.ByNamespace(ref.Namespace).Get(ref.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for %+v: %v", gvr, err)
	}
	serviceable, ok := obj.(*iduckv1alpha1.ServiceableType)
	if !ok {
		return nil, fmt.Errorf("%+v (%T) is not an ServiceableType", ref, ref)
	}
	return &serviceable.Status.Binding, nil
}
