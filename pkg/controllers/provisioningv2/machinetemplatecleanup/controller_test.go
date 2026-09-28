package machinetemplatecleanup

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	provv1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
// 	wfake "github.com/rancher/wrangler/v3/pkg/generic/fake"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.uber.org/mock/gomock"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/apimachinery/pkg/runtime/schema"
// 	k8sTypes "k8s.io/apimachinery/pkg/types"
// 	dynamicfake "k8s.io/client-go/dynamic/fake"
// 	capi "sigs.k8s.io/cluster-api/api/core/v1beta2"
// )

// type testEnv struct {
// 	ctrl             *gomock.Controller
// 	capiClusterCache *wfake.MockCacheInterface[*capi.Cluster]
// 	provClusterCache *wfake.MockCacheInterface[*provv1.Cluster]
// 	dynamicClient    *dynamicfake.FakeDynamicClient
// 	handler          *handler
// }

// func setupTestEnv(t *testing.T) *testEnv {
// 	t.Helper()

// 	ctrl := gomock.NewController(t)
// 	capiCache := wfake.NewMockCacheInterface[*capi.Cluster](ctrl)
// 	provCache := wfake.NewMockCacheInterface[*provv1.Cluster](ctrl)
// 	scheme := runtime.NewScheme()
// 	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)

// 	h := &handler{
// 		ctx:              context.Background(),
// 		capiClusterCache: capiCache,
// 		provClusterCache: provCache,
// 	}

// 	return &testEnv{
// 		ctrl:             ctrl,
// 		capiClusterCache: capiCache,
// 		provClusterCache: provCache,
// 		dynamicClient:    dynClient,
// 		handler:          h,
// 	}
// }

// func newUnstructuredResource(apiVersion, kind, namespace, name string, age time.Duration) *unstructured.Unstructured {
// 	obj := &unstructured.Unstructured{
// 		Object: map[string]interface{}{
// 			"apiVersion": apiVersion,
// 			"kind":       kind,
// 			"metadata": map[string]interface{}{
// 				"name":            name,
// 				"namespace":       namespace,
// 				"uid":             "uid-1234",
// 				"resourceVersion": "1",
// 			},
// 		},
// 	}
// 	obj.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-age)))
// 	return obj
// }

// func Test_cleanupInfraCluster(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	capiCache := wfake.NewMockCacheInterface[*capi.Cluster](ctrl)
// 	provCache := wfake.NewMockCacheInterface[*provv1.Cluster](ctrl)
// 	scheme := runtime.NewScheme()

// 	h := &handler{
// 		ctx:              context.Background(),
// 		capiClusterCache: capiCache,
// 		provClusterCache: provCache,
// 	}

// 	now, err := time.Parse(time.RFC3339, "2026-09-30T15:30:00Z")
// 	require.NoError(t, err)

// 	tests := []struct {
// 		name          string
// 		infraCluster  unstructured.Unstructured
// 		expectDeleted bool
// 	}{
// 		{
// 			name: "InfraCluster has no owners but is too recent",
// 			infraCluster: unstructured.Unstructured{
// 				Object: map[string]interface{}{
// 					"apiVersion": schema.GroupVersion{
// 						Group:   capi.GroupVersionInfrastructure.Group,
// 						Version: "v1beta2",
// 					}.String(),
// 					"kind": "AWSCluster",
// 					"metadata": map[string]interface{}{
// 						"name":              "foocluster",
// 						"namespace":         "fleet-default",
// 						"uid":               "uid-1234",
// 						"resourceVersion":   "1",
// 						"creationTimestamp": "2026-09-30T15:00:00Z",
// 					},
// 				},
// 			},
// 			expectDeleted: false,
// 		},
// 		{
// 			name: "InfraCluster has owners",
// 			infraCluster: unstructured.Unstructured{
// 				Object: map[string]interface{}{
// 					"apiVersion": schema.GroupVersion{
// 						Group:   capi.GroupVersionInfrastructure.Group,
// 						Version: "v1beta2",
// 					}.String(),
// 					"kind": "AWSCluster",
// 					"metadata": map[string]interface{}{
// 						"name":              "foocluster",
// 						"namespace":         "fleet-default",
// 						"uid":               "uid-1234",
// 						"resourceVersion":   "1",
// 						"creationTimestamp": "2026-09-30T14:00:00Z",
// 						"ownerReferences": []interface{}{
// 							map[string]interface{}{
// 								"apiVersion": "cluster.x-k8s.io/v1beta2",
// 								"kind":       "Cluster",
// 								"name":       "owner-cluster",
// 								"uid":        "owner-uid",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			expectDeleted: false,
// 		},
// 		{
// 			name: "InfraCluster has no owners and is sufficiently old",
// 			infraCluster: unstructured.Unstructured{
// 				Object: map[string]interface{}{
// 					"apiVersion": schema.GroupVersion{
// 						Group:   capi.GroupVersionInfrastructure.Group,
// 						Version: "v1beta2",
// 					}.String(),
// 					"kind": "AWSCluster",
// 					"metadata": map[string]interface{}{
// 						"name":              "foocluster",
// 						"namespace":         "fleet-default",
// 						"uid":               "uid-1234",
// 						"resourceVersion":   "1",
// 						"creationTimestamp": "2026-09-30T14:29:00Z",
// 					},
// 				},
// 			},
// 			expectDeleted: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

// 			infraClusterClient := dynamicClient.Resource(schema.GroupVersionResource{
// 				Group:    capi.GroupVersionInfrastructure.Group,
// 				Version:  "v1beta2",
// 				Resource: "awsclusters",
// 			})

// 			_, err = infraClusterClient.Namespace(
// 				tt.infraCluster.GetNamespace()).Create(t.Context(),
// 				&tt.infraCluster,
// 				metav1.CreateOptions{},
// 			)
// 			require.NoError(t, nil)

// 			// Clear create call.
// 			dynamicClient.ClearActions()

// 			err := h.cleanupInfraCluster2(t.Context(), &tt.infraCluster, infraClusterClient, now)
// 			require.NoError(t, err)

// 			if tt.expectDeleted {
// 				require.Len(t, dynamicClient.Actions(), 1)
// 				assert.Equal(t, "delete", dynamicClient.Actions()[0].GetVerb())
// 			} else {
// 				assert.Len(t, dynamicClient.Actions(), 0)
// 			}
// 		})
// 	}
// }

// func Test_cleanupInfraMachineTemplate3(t *testing.T) {
// }

// func Test_cleanupInfraMachineTemplate2(t *testing.T) {
// 	templateGK := schema.GroupKind{
// 		Group: capi.GroupVersionInfrastructure.Group,
// 		Kind:  "AWSMachineTemplate",
// 	}
// 	templateGVR := schema.GroupVersionResource{
// 		Group:    capi.GroupVersionInfrastructure.Group,
// 		Version:  "v1beta2",
// 		Resource: "awsmachinetemplates",
// 	}

// 	t.Run("skips template when within grace period", func(t *testing.T) {
// 		env := setupTestEnv(t)
// 		template := newUnstructuredResource("infrastructure.cluster.x-k8s.io/v1beta2", "AWSMachineTemplate", "default", "template-recent", 10*time.Minute)
// 		client := env.dynamicClient.Resource(templateGVR)
// 		cache := make(map[k8sTypes.UID]map[string]bool)

// 		err := env.handler.cleanupInfraMachineTemplate2(context.Background(), template, templateGK, cache, client, time.Now())
// 		require.NoError(t, err)
// 		assert.Empty(t, env.dynamicClient.Actions())
// 	})

// 	t.Run("deletes unadopted template after grace period", func(t *testing.T) {
// 		env := setupTestEnv(t)
// 		template := newUnstructuredResource("infrastructure.cluster.x-k8s.io/v1beta2", "AWSMachineTemplate", "default", "template-unadopted", 2*templateGracePeriod)
// 		client := env.dynamicClient.Resource(templateGVR)

// 		_, err := client.Namespace(template.GetNamespace()).Create(context.Background(), template, metav1.CreateOptions{})
// 		require.NoError(t, err)
// 		env.dynamicClient.ClearActions()

// 		cache := make(map[k8sTypes.UID]map[string]bool)
// 		err = env.handler.cleanupInfraMachineTemplate2(context.Background(), template, templateGK, cache, client, time.Now())
// 		require.NoError(t, err)

// 		actions := env.dynamicClient.Actions()
// 		require.Len(t, actions, 1)
// 		assert.Equal(t, "delete", actions[0].GetVerb())
// 	})

// 	t.Run("returns error when template has more than one owner reference", func(t *testing.T) {
// 		env := setupTestEnv(t)
// 		template := newUnstructuredResource("infrastructure.cluster.x-k8s.io/v1beta2", "AWSMachineTemplate", "default", "template-multi-owner", 2*templateGracePeriod)
// 		template.SetOwnerReferences([]metav1.OwnerReference{
// 			{
// 				APIVersion: "cluster.x-k8s.io/v1beta2",
// 				Kind:       "Cluster",
// 				Name:       "owner-1",
// 				UID:        "uid-1",
// 			},
// 			{
// 				APIVersion: "cluster.x-k8s.io/v1beta2",
// 				Kind:       "Cluster",
// 				Name:       "owner-2",
// 				UID:        "uid-2",
// 			},
// 		})
// 		client := env.dynamicClient.Resource(templateGVR)
// 		cache := make(map[k8sTypes.UID]map[string]bool)

// 		err := env.handler.cleanupInfraMachineTemplate2(context.Background(), template, templateGK, cache, client, time.Now())
// 		require.ErrorContains(t, err, "has too many owners")
// 	})
// }
