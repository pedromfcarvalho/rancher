// selfuser implements the store for the imperative selfuser resource.
package kdmrelease

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/rancher/channelserver/pkg/model"
	ext "github.com/rancher/rancher/pkg/apis/ext.cattle.io/v1"
	"github.com/rancher/rancher/pkg/channelserver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/kubernetes/pkg/printers"
	printerstorage "k8s.io/kubernetes/pkg/printers/storage"
)

const (
	SingularName = "kdmrelease"
	PluralName   = "kdmreleases"
	kind         = "KDMRelease"
	DistroLabel  = "cattle.io/distribution"
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
	return &ext.KDMRelease{}
}

func (s *Store) Destroy() {
}

func (s *Store) NewList() runtime.Object {
	return &ext.KDMReleaseList{}
}

type contToken struct {
	Index int    `json:"i"`
	Hash  string `json:"h"`
}

func (s *Store) List(
	ctx context.Context,
	options *metainternalversion.ListOptions,
) (runtime.Object, error) {
	releases := channelserver.GetVersionedReleases()

	if releases == nil {
		return nil, apierrors.NewServiceUnavailable("KDM data is still being loaded")
	}

	items := make([]ext.KDMRelease, 0, len(releases.K3S.Releases)+len(releases.RKE2.Releases))

	for _, release := range releases.K3S.Releases {
		r := convertRelease(&release, "k3s", releases.Hash)

		if options.LabelSelector == nil {
			items = append(items, *r)
		} else if options.LabelSelector.Matches(labels.Set(r.ObjectMeta.Labels)) {
			items = append(items, *r)
		}
	}

	for _, release := range releases.RKE2.Releases {
		r := convertRelease(&release, "rke2", releases.Hash)

		if options.LabelSelector == nil {
			items = append(items, *r)
		} else if options.LabelSelector.Matches(labels.Set(r.ObjectMeta.Labels)) {
			items = append(items, *r)
		}
	}

	var idx int
	if options.Continue != "" {
		var cont contToken

		contBytes, err := base64.StdEncoding.DecodeString(options.Continue)
		if err != nil {
			return nil, apierrors.NewBadRequest("could not decode continue token")
		}

		err = json.Unmarshal(contBytes, &cont)
		if err != nil {
			return nil, apierrors.NewBadRequest("could not decode continue token")
		}

		if cont.Hash != releases.Hash {
			return nil, apierrors.NewResourceExpired("releases version not available")
		}

		idx = cont.Index
	}

	var cont string

	limit := int(options.Limit)
	if limit != 0 {
		if idx+limit > len(items) {
			items = items[idx:]
		} else {
			items = items[idx : idx+limit]
			idx = idx + limit

			contBytes, err := json.Marshal(contToken{
				Index: idx,
				Hash:  releases.Hash,
			})
			if err != nil {
				return nil, apierrors.NewInternalError(err)
			}

			cont = base64.StdEncoding.EncodeToString(contBytes)
		}
	}

	return &ext.KDMReleaseList{
		ListMeta: metav1.ListMeta{
			Continue:        cont,
			ResourceVersion: "1233456788",
		},
		Items: items,
	}, nil
}

func (s *Store) Get(
	ctx context.Context,
	name string,
	options *metav1.GetOptions,
) (runtime.Object, error) {
	releases := channelserver.GetVersionedReleases()

	if releases == nil {
		return nil, apierrors.NewServiceUnavailable("KDM data is still being loaded")
	}

	for _, release := range releases.K3S.Releases {
		if versionToName(release.Version) == name {
			return convertRelease(&release, "k3s", releases.Hash), nil
		}
	}

	for _, release := range releases.RKE2.Releases {
		if versionToName(release.Version) == name {
			return convertRelease(&release, "rke2", releases.Hash), nil
		}
	}

	return nil, apierrors.NewNotFound(GVR.GroupResource(), name)
}

func (t *Store) ConvertToTable(
	ctx context.Context,
	object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return t.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func versionToName(version string) string {
	return strings.ReplaceAll(version, "+", "-")
}

func convertRelease(rel *model.Release, distribution string, hash string) *ext.KDMRelease {
	conv := &ext.KDMRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name: versionToName(rel.Version),
			Labels: map[string]string{
				DistroLabel: distribution,
			},
			ResourceVersion: hash + "-" + rel.Version,
		},
		Spec: ext.KDMReleaseSpec{
			Release: ext.Release{
				Version:                 rel.Version,
				Distribution:            distribution,
				ChannelServerMinVersion: rel.ChannelServerMinVersion,
				ChannelServerMaxVersion: rel.ChannelServerMaxVersion,
				ServerArgs:              make(map[string]ext.KDMReleaseField, len(rel.ServerArgs)),
				AgentArgs:               make(map[string]ext.KDMReleaseField, len(rel.AgentArgs)),
				FeatureVersions:         make(map[string]string, len(rel.FeatureVersions)),
				Charts:                  make(map[string]ext.KDMReleaseChart, len(rel.Charts)),
			},
		},
	}

	for k, v := range rel.ServerArgs {
		conv.Spec.ServerArgs[k] = ext.KDMReleaseField{
			Type:     v.Type,
			Options:  v.Options,
			Default:  v.Default,
			Nullable: v.Nullable,
		}
	}

	for k, v := range rel.AgentArgs {
		conv.Spec.AgentArgs[k] = ext.KDMReleaseField{
			Type:     v.Type,
			Options:  v.Options,
			Default:  v.Default,
			Nullable: v.Nullable,
		}
	}

	for k, v := range rel.FeatureVersions {
		conv.Spec.FeatureVersions[k] = v
	}

	for k, v := range rel.Charts {
		conv.Spec.Charts[k] = ext.KDMReleaseChart{
			Repo:    v.Repo,
			Version: v.Version,
		}
	}

	return conv
}

func printHandler(h printers.PrintHandler) {
	columnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Version", Type: "string", Description: "kubernetes release version"},
		{Name: "Disribution", Type: "string", Description: "kubernetes distribution"},
	}
	_ = h.TableHandler(columnDefinitions, printReleaseList)
	_ = h.TableHandler(columnDefinitions, printRelease)
}

func printRelease(release *ext.KDMRelease, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	return []metav1.TableRow{
		{
			Object: runtime.RawExtension{Object: release},
			Cells: []any{
				release.Name,
				release.Spec.Version,
				release.Spec.Distribution,
			},
		},
	}, nil
}

func printReleaseList(releaseList *ext.KDMReleaseList, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	rows := make([]metav1.TableRow, 0, len(releaseList.Items))
	for i := range releaseList.Items {
		r, err := printRelease(&releaseList.Items[i], options)
		if err != nil {
			return nil, err
		}
		rows = append(rows, r...)
	}
	return rows, nil
}
