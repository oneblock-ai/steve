package cluster

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oneblock-ai/apiserver/v2/pkg/store/empty"
	"github.com/oneblock-ai/apiserver/v2/pkg/types"
	detector "github.com/rancher/kubernetes-provider-detector"
	"github.com/rancher/wrangler/v2/pkg/genericcondition"
	"github.com/rancher/wrangler/v2/pkg/schemas"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"

	"github.com/oneblock-ai/steve/v2/pkg/accesscontrol"
	"github.com/oneblock-ai/steve/v2/pkg/attributes"
	steveschema "github.com/oneblock-ai/steve/v2/pkg/schema"
	"github.com/oneblock-ai/steve/v2/pkg/stores/proxy"
)

const (
	clusterApiGroup       = "management.oneblock.ai"
	clusterApiVersion     = "v1"
	clusterKindName       = "Cluster"
	clusterSchemaTypeName = "management.oneblock.ai.cluster"
)

func Register(ctx context.Context, apiSchemas *types.APISchemas, cg proxy.ClientGetter, schemaFactory steveschema.Factory) {
	apiSchemas.InternalSchemas.TypeName(clusterSchemaTypeName, Cluster{})

	apiSchemas.MustImportAndCustomize(&ApplyInput{}, nil)
	apiSchemas.MustImportAndCustomize(&ApplyOutput{}, nil)
	apiSchemas.MustImportAndCustomize(Cluster{}, func(schema *types.APISchema) {
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
		schema.Attributes["access"] = accesscontrol.AccessListByVerb{
			"watch": accesscontrol.AccessList{
				{
					Namespace:    "*",
					ResourceName: "*",
				},
			},
		}
		schema.Store = &Store{
			provider:  provider(ctx, cg),
			discovery: discoveryClient(cg),
		}
		attributes.SetGVK(schema, schema2.GroupVersionKind{
			Group:   clusterApiGroup,
			Version: clusterApiVersion,
			Kind:    clusterKindName,
		})

		schema.ActionHandlers = map[string]http.Handler{
			"apply": &Apply{
				cg:            cg,
				schemaFactory: schemaFactory,
			},
		}
		schema.ResourceActions = map[string]schemas.Action{
			"apply": {
				Input:  "applyInput",
				Output: "applyOutput",
			},
		}
	})
}

func discoveryClient(cg proxy.ClientGetter) discovery.DiscoveryInterface {
	k8s, err := cg.AdminK8sInterface()
	if err != nil {
		return nil
	}
	return k8s.Discovery()
}

func provider(ctx context.Context, cg proxy.ClientGetter) string {
	var (
		provider string
		err      error
	)

	k8s, err := cg.AdminK8sInterface()
	if err == nil {
		provider, _ = detector.DetectProvider(ctx, k8s)
	}

	return provider
}

func AddApply(apiSchemas *types.APISchemas, schema *types.APISchema) {
	if _, ok := schema.ActionHandlers["apply"]; ok {
		return
	}
	cluster := apiSchemas.LookupSchema(clusterSchemaTypeName)
	if cluster == nil {
		return
	}

	actionHandler, ok := cluster.ActionHandlers["apply"]
	if !ok {
		return
	}

	if schema.ActionHandlers == nil {
		schema.ActionHandlers = map[string]http.Handler{}
	}
	schema.ActionHandlers["apply"] = actionHandler

	if schema.ResourceActions == nil {
		schema.ResourceActions = map[string]schemas.Action{}
	}
	schema.ResourceActions["apply"] = schemas.Action{
		Input:  "applyInput",
		Output: "applyOutput",
	}
}

type Store struct {
	empty.Store
	provider  string
	discovery discovery.DiscoveryInterface
}

func (s *Store) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	if apiOp.Namespace == "" && id == "local" {
		return s.getLocal(), nil
	}
	return s.Store.ByID(apiOp, schema, id)
}

func (s *Store) getLocal() types.APIObject {
	var (
		info *version.Info
	)

	if s.discovery != nil {
		info, _ = s.discovery.ServerVersion()
	}
	return types.APIObject{
		ID: "local",
		Object: &Cluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       clusterKindName,
				APIVersion: fmt.Sprintf("%s/%s", clusterApiGroup, clusterApiVersion),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "local",
			},
			Spec: Spec{
				DisplayName: "Local Cluster",
				Internal:    true,
			},
			Status: Status{
				Version:  info,
				Driver:   "local",
				Provider: s.provider,
				Conditions: []genericcondition.GenericCondition{
					{
						Type:   "Ready",
						Status: "True",
					},
				},
			},
		},
	}

}

func (s *Store) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	if apiOp.Namespace != "" {
		return s.Store.List(apiOp, schema)
	}

	return types.APIObjectList{
		Objects: []types.APIObject{
			s.getLocal(),
		},
	}, nil
}

func (s *Store) Watch(apiOp *types.APIRequest, schema *types.APISchema, w types.WatchRequest) (chan types.APIEvent, error) {
	result := make(chan types.APIEvent, 1)
	result <- types.APIEvent{
		Name:         "local",
		ResourceType: "management.oneblock.ai.clusters",
		ID:           "local",
		Object:       s.getLocal(),
	}

	go func() {
		<-apiOp.Context().Done()
		close(result)
	}()

	return result, nil
}
