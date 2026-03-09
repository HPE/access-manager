/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package accessmanager

import (
	"context"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"strings"

	"github.com/hpe/access-manager/internal/services/metadata"
	"github.com/hpe/access-manager/pkg/logger"
	"github.com/hpe/access-manager/pkg/metrics"
	"github.com/hpe/access-manager/pkg/middleware"
)

/*
The ConcreteAccessManager is the entry point for the overall Access Manager from
either the public GRPC interface or from the REST interface.

All that should happen here is to handle the incoming requests and forward
them on to the underlying PermissionLogic instance. Neither permission nor
version checking should be done at this level.
*/
type ConcreteAccessManager struct {
	UnimplementedAccessManagerServer
	appMetrics *metrics.Metrics
	meta       PermissionLogic
}

func NewAccessManager(meta PermissionLogic, appMetrics *metrics.Metrics) *ConcreteAccessManager {
	return &ConcreteAccessManager{meta: meta, appMetrics: appMetrics}
}

func (p *ConcreteAccessManager) Bootstrap(ctx context.Context, request *BoostrapRequest) (*BootstrapResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.Path: request.Boot,
	}).Msg("bootstrap")

	err := p.meta.Bootstrap(request.Boot, request.Key)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &BootstrapResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &BootstrapResponse{
		Error: &Status{
			Error:   0,
			Message: "",
		},
	}, nil
}

func (p *ConcreteAccessManager) CreateObject(ctx context.Context, request *CreateObjectRequest) (*CreateObjectResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     request.Path,
	}).Msg("create-object")
	path := updatePathPrefix(request.Path)

	var ax metadata.Annotation
	if request.AsDirectory {
		ax = metadata.Annotation{Tag: "dir"}
	} else {
		ax = metadata.Annotation{Tag: "leaf"}
	}
	err := p.meta.CreateObject(ctx, path, request.CallerId, &ax)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &CreateObjectResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, err
	}
	return &CreateObjectResponse{
		Error: &Status{
			Error:   0,
			Message: "",
		},
	}, err
}

func (p *ConcreteAccessManager) Annotate(ctx context.Context, request *AnnotateRequest) (*AnnotateResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	path := updatePathPrefix(request.Path)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     path,
	}).Msg("annotate")

	ann, _, err := metadata.UnmarshalAnnotation([]byte(request.Annotation))
	if err != nil {
		return nil, err
	}
	if err := p.meta.Annotate(ctx, path, ann, request.CallerId); err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &AnnotateResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &AnnotateResponse{
		Error: &Status{
			Error:   0,
			Message: "",
		},
	}, nil
}

func (p *ConcreteAccessManager) DeleteAnnotation(ctx context.Context, request *DeleteAnnotationRequest) (*DeleteAnnotationResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	path := updatePathPrefix(request.Path)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     path,
	}).Msg("delete-annotation")
	if err := p.meta.DeleteAnnotation(ctx, path, request.Tag, request.Unique, request.CallerId); err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &DeleteAnnotationResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, err
	}
	return &DeleteAnnotationResponse{
		Error: &Status{
			Error:   0,
			Message: "",
		},
	}, nil
}

func (p *ConcreteAccessManager) GetPrincipalCredential(
	ctx context.Context,
	request *GetPrincipalCredentialRequest,
) (*GetPrincipalCredentialResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     request.Path,
	}).Msg("get-principal-credential")
	cred, err := p.meta.GetPrincipalCredential(ctx, request.Path, request.CallerId)
	fmt.Printf("cred = %s\n", cred)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &GetPrincipalCredentialResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &GetPrincipalCredentialResponse{
		Credential: cred,
		Error: &Status{
			Error: 0,
		},
	}, nil
}

func (p *ConcreteAccessManager) DeleteObject(
	ctx context.Context,
	request *DeleteObjectRequest,
) (*DeleteObjectResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     request.Path,
	}).Msg("delete-object")

	path := updatePathPrefix(request.Path)
	err := p.meta.DeleteObject(ctx, path, false, request.CallerId)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &DeleteObjectResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &DeleteObjectResponse{
		Error: &Status{
			Error: 0,
		},
	}, nil
}

func (p *ConcreteAccessManager) GetDetails(ctx context.Context, request *GetDetailsRequest) (
	*GetDetailsResponse,
	error,
) {
	defer p.appMetrics.AddGrpcReqLatency()()

	ctx = middleware.AddRequestIDToContext(ctx)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID:        request.CallerId,
		logger.Path:            request.Path,
		logger.IncludeChildren: request.IncludeChildren,
	}).Msg("getting details")

	path := updatePathPrefix(request.Path)

	details, children, err := p.meta.GetDetails(ctx, path, request.IncludeChildren, request.CallerId)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &GetDetailsResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}

	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     path,
	}).Msg("got details")

	return &GetDetailsResponse{
		Details:  details,
		Children: children,
		Error: &Status{
			Error: 0,
		},
	}, nil
}

// special case handling for the REST interface where the path is embedded in the URL
// in that case the am:// part will be missing.
func updatePathPrefix(path string) string {
	if !strings.HasPrefix(path, common.StandardPrefix) {
		if path == "" || strings.HasPrefix(path, "user") || strings.HasPrefix(path, "workload") ||
			strings.HasPrefix(path, "role") || strings.HasPrefix(path, "data") {
			path = common.StandardPrefix + path
		} else if strings.HasPrefix(path, "am/") {
			path = strings.Replace(path, "am/", "am://", 1)
		}
	}

	if path != common.StandardPrefix && strings.HasSuffix(path, "/") {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func (p *ConcreteAccessManager) GetDatasetCredential(
	ctx context.Context,
	request *GetDatasetCredentialRequest) (*GetDatasetCredentialResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
		logger.Path:     request.Path,
	}).Msg("")

	operations := make([]metadata.Operation, len(request.Operations))
	for _, op := range request.Operations {
		operations = append(operations, op)
	}

	// TODO fix credential building
	credential, _, err := p.meta.GetDatasetCredential(ctx, request.Path, operations, request.CallerId)
	if err != nil {
		return &GetDatasetCredentialResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}

	return &GetDatasetCredentialResponse{
		Url:        "fix me",
		Credential: credential,
		Error: &Status{
			Error: 0,
		},
	}, nil
}

func (p *ConcreteAccessManager) GetSigningKeys(
	ctx context.Context,
	request *GetSigningKeysRequest) (*GetSigningKeysResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)

	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
	})
	keys, err := p.meta.GetSigningKeys(ctx, request.CallerId)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &GetSigningKeysResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &GetSigningKeysResponse{
		Keys: keys,
		Error: &Status{
			Error: 0,
		},
	}, nil
}

func (p *ConcreteAccessManager) ValidateCredential(
	ctx context.Context,
	request *ValidateCredentialRequest) (*ValidateCredentialResponse, error) {
	defer p.appMetrics.AddGrpcReqLatency()()
	ctx = middleware.AddRequestIDToContext(ctx)
	logger.GetLogger().Info().Ctx(ctx).Fields(map[string]any{
		logger.CallerID: request.CallerId,
	})
	claims, err := p.meta.ValidateCredential(ctx, request.Credential, request.CallerId)
	if err != nil {
		p.appMetrics.AddGrpcErrorCount()
		return &ValidateCredentialResponse{
			Error: &Status{
				Error:   1,
				Message: err.Error(),
			},
		}, nil
	}
	return &ValidateCredentialResponse{
		Info:  claims,
		Valid: true,
		Error: &Status{
			Error:   0,
			Message: "credential is valid",
		},
	}, nil
}
