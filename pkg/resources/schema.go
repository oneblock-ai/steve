package resources

import (
	"context"

	"github.com/oneblock-ai/apiserver/v2/pkg/store/apiroot"
	"github.com/oneblock-ai/apiserver/v2/pkg/subscribe"
	"github.com/oneblock-ai/apiserver/v2/pkg/types"
	corecontrollers "github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/discovery"

	"github.com/oneblock-ai/steve/v2/pkg/accesscontrol"
	"github.com/oneblock-ai/steve/v2/pkg/client"
	"github.com/oneblock-ai/steve/v2/pkg/clustercache"
	"github.com/oneblock-ai/steve/v2/pkg/resources/apigroups"
	"github.com/oneblock-ai/steve/v2/pkg/resources/cluster"
	"github.com/oneblock-ai/steve/v2/pkg/resources/common"
	"github.com/oneblock-ai/steve/v2/pkg/resources/counts"
	"github.com/oneblock-ai/steve/v2/pkg/resources/formatters"
	"github.com/oneblock-ai/steve/v2/pkg/resources/userpreferences"
	"github.com/oneblock-ai/steve/v2/pkg/schema"
	steveschema "github.com/oneblock-ai/steve/v2/pkg/schema"
	"github.com/oneblock-ai/steve/v2/pkg/stores/proxy"
	"github.com/oneblock-ai/steve/v2/pkg/summarycache"
)

func DefaultSchemas(ctx context.Context, baseSchema *types.APISchemas, ccache clustercache.ClusterCache,
	cg proxy.ClientGetter, schemaFactory steveschema.Factory, serverVersion string) error {
	counts.Register(baseSchema, ccache)
	subscribe.Register(baseSchema, func(apiOp *types.APIRequest) *types.APISchemas {
		user, ok := request.UserFrom(apiOp.Context())
		if ok {
			schemas, err := schemaFactory.Schemas(user)
			if err == nil {
				return schemas
			}
		}
		return apiOp.Schemas
	}, serverVersion)
	apiroot.Register(baseSchema, []string{"v1"}, "proxy:/apis")
	cluster.Register(ctx, baseSchema, cg, schemaFactory)
	userpreferences.Register(baseSchema)
	return nil
}

func DefaultSchemaTemplates(cf *client.Factory,
	baseSchemas *types.APISchemas,
	summaryCache *summarycache.SummaryCache,
	lookup accesscontrol.AccessSetLookup,
	discovery discovery.DiscoveryInterface,
	namespaceCache corecontrollers.NamespaceCache) []schema.Template {
	return []schema.Template{
		common.DefaultTemplate(cf, summaryCache, lookup, namespaceCache),
		apigroups.Template(discovery),
		{
			ID:        "configmap",
			Formatter: formatters.DropHelmData,
		},
		{
			ID:        "secret",
			Formatter: formatters.DropHelmData,
		},
		{
			ID:        "pod",
			Formatter: formatters.Pod,
		},
		{
			ID: "management.cattle.io.cluster",
			Customize: func(apiSchema *types.APISchema) {
				cluster.AddApply(baseSchemas, apiSchema)
			},
		},
	}
}
