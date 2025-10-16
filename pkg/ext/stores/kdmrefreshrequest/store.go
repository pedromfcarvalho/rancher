// selfuser implements the store for the imperative selfuser resource.
package kdmrefreshrequest

import (
	"context"
	"fmt"

	ext "github.com/rancher/rancher/pkg/apis/ext.cattle.io/v1"
	kd "github.com/rancher/rancher/pkg/kontainerdrivermetadata"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

const (
	SingularName = "kdmrefreshrequest"
	kind         = "KDMRefreshRequest"
)

var (
	_ rest.Creater                  = &Store{}
	_ rest.Storage                  = &Store{}
	_ rest.Scoper                   = &Store{}
	_ rest.SingularNameProvider     = &Store{}
	_ rest.GroupVersionKindProvider = &Store{}
)

var GVK = ext.SchemeGroupVersion.WithKind(kind)

// +k8s:openapi-gen=false
// +k8s:deepcopy-gen=false

type Store struct {
	metadataHandler kd.MetadataController
}

func New(metadataHandler kd.MetadataController) *Store {
	return &Store{
		metadataHandler: metadataHandler,
	}
}

func (s *Store) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {

	kdmRefreshRequest, ok := obj.(*ext.KDMRefreshRequest)
	if !ok {
		var zeroT *ext.KDMRefreshRequest
		return nil, apierrors.NewInternalError(fmt.Errorf("expected %T but got %T",
			zeroT, obj))
	}

	var err error
	if kdmRefreshRequest.Spec.Wait {
		err = s.metadataHandler.RefreshSync(ctx)
	} else {
		err = s.metadataHandler.Refresh()
	}

	if err != nil {
		logrus.Errorf("refreshing KDM from KDMRefreshRequest: %v", err)
		return nil, apierrors.NewInternalError(fmt.Errorf("could not refresh KDM"))
	}

	return kdmRefreshRequest, nil
}

func (s *Store) GroupVersionKind(_ schema.GroupVersion) schema.GroupVersionKind {
	return GVK
}

func (s *Store) NamespaceScoped() bool {
	return false
}

func (s *Store) GetSingularName() string {
	return SingularName
}

func (s *Store) New() runtime.Object {
	return &ext.KDMRefreshRequest{}
}

func (s *Store) Destroy() {
}
