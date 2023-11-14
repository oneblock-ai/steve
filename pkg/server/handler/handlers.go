package handler

import (
	"github.com/gorilla/mux"
	"github.com/oneblock-ai/apiserver/v2/pkg/types"

	"github.com/oneblock-ai/steve/v2/pkg/attributes"
	"github.com/oneblock-ai/steve/v2/pkg/schema"
)

func k8sAPI(sf schema.Factory, apiOp *types.APIRequest) {
	vars := mux.Vars(apiOp.Request)
	apiOp.Name = vars["name"]
	apiOp.Type = vars["type"]

	nOrN := vars["nameorns"]
	if nOrN != "" {
		schema := apiOp.Schemas.LookupSchema(apiOp.Type)
		if attributes.Namespaced(schema) {
			vars["namespace"] = nOrN
		} else {
			vars["name"] = nOrN
		}
	}

	if namespace := vars["namespace"]; namespace != "" {
		apiOp.Namespace = namespace
	}
}

func apiRoot(sf schema.Factory, apiOp *types.APIRequest) {
	apiOp.Type = "apiRoot"
}
