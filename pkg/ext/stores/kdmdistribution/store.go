// selfuser implements the store for the imperative selfuser resource.
package kdmdistribution

import (
	"context"

	"github.com/rancher/channelserver/pkg/model"
	ext "github.com/rancher/rancher/pkg/apis/ext.cattle.io/v1"
	"github.com/rancher/rancher/pkg/channelserver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/kubernetes/pkg/printers"
	printerstorage "k8s.io/kubernetes/pkg/printers/storage"
)

const (
	SingularName = "kdmdistribution"
	PluralName   = "kdmdistributions"
	kind         = "KDMDistribution"
)

var (
	_ rest.Storage                  = &Store{}
	_ rest.Scoper                   = &Store{}
	_ rest.SingularNameProvider     = &Store{}
	_ rest.GroupVersionKindProvider = &Store{}
	_ rest.Lister                   = &Store{}
	_ rest.Getter                   = &Store{}
)

var GVK = ext.SchemeGroupVersion.WithKind(kind)
var GVR = ext.SchemeGroupVersion.WithResource(PluralName)

// +k8s:openapi-gen=false
// +k8s:deepcopy-gen=false

type Store struct {
	tableConverter rest.TableConvertor
}

func New() *Store {
	return &Store{
		tableConverter: printerstorage.TableConvertor{
			TableGenerator: printers.NewTableGenerator().With(printHandler),
		},
	}
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
	return &ext.KDMDistribution{}
}

func (s *Store) Destroy() {
}

func (s *Store) NewList() runtime.Object {
	return &ext.KDMDistributionList{}
}

func (s *Store) Get(
	ctx context.Context,
	name string,
	options *metav1.GetOptions,
) (runtime.Object, error) {

	obj := s.get(name)
	if obj == nil {
		return nil, apierrors.NewNotFound(GVR.GroupResource(), name)
	}

	return obj, nil
}

func (s *Store) get(name string) *ext.KDMDistribution {
	cfg := channelserver.GetReleaseConfigByRuntime(context.TODO(), name)
	if cfg == nil {
		return nil
	}

	releases := make([]ext.Release, 0, len(cfg.ReleasesConfig().Releases))

	for _, release := range cfg.ReleasesConfig().Releases {
		releases = append(releases, *convertRelease(&release, name))
	}

	return &ext.KDMDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: ext.KDMDistributionSpec{
			Releases: releases,
		},
	}
}

func (s *Store) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	k3s := s.get("k3s")
	rke2 := s.get("rke2")

	return &ext.KDMDistributionList{
		Items: []ext.KDMDistribution{
			*k3s,
			*rke2,
		},
	}, nil
}

func (t *Store) ConvertToTable(
	ctx context.Context,
	object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return t.tableConverter.ConvertToTable(ctx, object, tableOptions)
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

func printHandler(h printers.PrintHandler) {
	columnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Disribution", Type: "string", Description: "kubernetes distribution"},
	}
	_ = h.TableHandler(columnDefinitions, printDistributionList)
	_ = h.TableHandler(columnDefinitions, printDistribution)
}

func printDistribution(distribution *ext.KDMDistribution, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	return []metav1.TableRow{
		{
			Object: runtime.RawExtension{Object: distribution},
			Cells: []any{
				distribution.Name,
			},
		},
	}, nil
}

func printDistributionList(releaseList *ext.KDMDistributionList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(releaseList.Items))
	for i := range releaseList.Items {
		r, err := printDistribution(&releaseList.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}
