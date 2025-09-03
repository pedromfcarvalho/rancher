// selfuser implements the store for the imperative selfuser resource.
package kdmrequest

import (
	"context"
	"fmt"

	"github.com/rancher/channelserver/pkg/model"
	ext "github.com/rancher/rancher/pkg/apis/ext.cattle.io/v1"
	"github.com/rancher/rancher/pkg/channelserver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

const (
	SingularName = "kdmrequest"
	kind         = "KDMRequest"
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

type Store struct{}

func New() *Store {
	return &Store{}
}

func (s *Store) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {

	kdmRequest, ok := obj.(*ext.KDMRequest)
	if !ok {
		var zeroT *ext.KDMRequest
		return nil, apierrors.NewInternalError(fmt.Errorf("expected %T but got %T",
			zeroT, obj))
	}

	var configReleases []model.Release

	if kdmRequest.Spec.Distribution != "" {
		if kdmRequest.Spec.Distribution == "k3s" || kdmRequest.Spec.Distribution == "rke2" {
			configReleases = channelserver.GetReleaseConfigByRuntime(context.TODO(), kdmRequest.Spec.Distribution).ReleasesConfig().Releases
		}
	} else {
		configReleases = channelserver.GetReleaseConfigByRuntime(context.TODO(), "k3s").ReleasesConfig().Releases
		configReleases = append(configReleases, channelserver.GetReleaseConfigByRuntime(context.TODO(), "rke2").ReleasesConfig().Releases...)
	}

	releases := make([]ext.Release, 0, len(configReleases))

	for _, release := range configReleases {
		releases = append(releases, *convertRelease(&release, kdmRequest.Spec.Distribution))
	}

	if kdmRequest.Spec.Version != "" {
		for _, release := range releases {
			if release.Version == kdmRequest.Spec.Version {
				kdmRequest.Status.Releases = []ext.Release{
					release,
				}
				break
			}
		}
	} else {
		kdmRequest.Status.Releases = releases
	}

	return kdmRequest, nil
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
	return &ext.KDMRequest{}
}

func (s *Store) Destroy() {
}

func convertRelease(rel *model.Release, distro string) *ext.Release {
	conv := &ext.Release{
		Distribution:            distro,
		Version:                 rel.Version,
		ChannelServerMinVersion: rel.ChannelServerMinVersion,
		ChannelServerMaxVersion: rel.ChannelServerMaxVersion,
		ServerArgs:              make(map[string]ext.KDMReleaseField, len(rel.ServerArgs)),
		AgentArgs:               make(map[string]ext.KDMReleaseField, len(rel.AgentArgs)),
		FeatureVersions:         make(map[string]string, len(rel.FeatureVersions)),
		Charts:                  make(map[string]ext.KDMReleaseChart, len(rel.Charts)),
	}

	for k, v := range rel.ServerArgs {
		conv.ServerArgs[k] = ext.KDMReleaseField{
			Type:     v.Type,
			Options:  v.Options,
			Default:  v.Default,
			Nullable: v.Nullable,
		}
	}

	for k, v := range rel.AgentArgs {
		conv.AgentArgs[k] = ext.KDMReleaseField{
			Type:     v.Type,
			Options:  v.Options,
			Default:  v.Default,
			Nullable: v.Nullable,
		}
	}

	for k, v := range rel.FeatureVersions {
		conv.FeatureVersions[k] = v
	}

	for k, v := range rel.Charts {
		conv.Charts[k] = ext.KDMReleaseChart{
			Repo:    v.Repo,
			Version: v.Version,
		}
	}

	return conv
}
