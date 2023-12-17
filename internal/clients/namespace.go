package clients

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	enums "go.temporal.io/api/enums/v1"
	ns "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

const (
	day = time.Hour * 24
)

type NamespaceService interface {
	DescribeNamespaceByName(ctx context.Context, name string) (*core.TemporalNamespaceObservation, error)

	CreateNamespace(ctx context.Context, namespace *core.TemporalNamespaceParameters) error
	UpdateNamespaceByName(ctx context.Context, namespace *core.TemporalNamespaceParameters) error
	DeleteNamespaceByName(ctx context.Context, name string) error

	MapObservationToNamespaceParameters(ns *core.TemporalNamespaceObservation) (*core.TemporalNamespaceParameters, error)
}

func (s *TemporalServiceImpl) MapObservationToNamespaceParameters(ns *core.TemporalNamespaceObservation) (*core.TemporalNamespaceParameters, error) {
	nsJson, err := json.Marshal(ns)
	if err != nil {
		return nil, err
	}

	var nsParam = core.TemporalNamespaceParameters{}
	err = json.Unmarshal(nsJson, &nsParam)
	if err != nil {
		return nil, err
	}

	return &nsParam, nil
}

func (s *TemporalServiceImpl) CreateNamespace(ctx context.Context, namespace *core.TemporalNamespaceParameters) error {
	retentionDuration := time.Duration(namespace.WorkflowExecutionRetentionDays) * day

	createrequest := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace.Name,
		Description:                      namespace.Description,
		OwnerEmail:                       namespace.OwnerEmail,
		WorkflowExecutionRetentionPeriod: &retentionDuration,
		Data:                             namespace.Data,
		HistoryArchivalState:             enums.ArchivalState(enums.ArchivalState_value[namespace.HistoryArchivalState]),
		HistoryArchivalUri:               namespace.HistoryArchivalUri,
		VisibilityArchivalState:          enums.ArchivalState(enums.ArchivalState_value[namespace.VisibilityArchivalState]),
		VisibilityArchivalUri:            namespace.VisibilityArchivalUri,
	}

	_, err := s.client.WorkflowService().RegisterNamespace(ctx, createrequest)
	var namespaceAlreadyExists *serviceerror.NamespaceAlreadyExists

	if errors.As(err, &namespaceAlreadyExists) {
		s.logger.Debug("Namespace '" + namespace.Name + "' already exists. " + err.Error())
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func (s *TemporalServiceImpl) DeleteAllNamespaces(ctx context.Context) error {
	namespaces, err := s.ListAllNamespaces(ctx)

	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		err := s.DeleteNamespaceByName(ctx, namespace.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TemporalServiceImpl) DescribeNamespaceByName(ctx context.Context, name string) (*core.TemporalNamespaceObservation, error) {
	request := &workflowservice.DescribeNamespaceRequest{
		Namespace: name,
	}

	response, err := s.client.WorkflowService().DescribeNamespace(ctx, request)

	var namespaceNotFound *serviceerror.NamespaceNotFound
	if errors.As(err, &namespaceNotFound) {
		s.logger.Debug("Namespace '" + name + "' not found. " + err.Error())
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, nil
	}

	return mapDescribeNamespaceResponse(response), nil
}

func (s *TemporalServiceImpl) DeleteNamespaceByName(ctx context.Context, name string) error {
	deleterequest := &operatorservice.DeleteNamespaceRequest{
		Namespace: name,
	}

	namespace, err := s.DescribeNamespaceByName(ctx, name)
	if namespace != nil {
		response, err := s.client.OperatorService().DeleteNamespace(ctx, deleterequest)

		var namespaceInvalidState *serviceerror.NamespaceInvalidState
		if errors.As(err, &namespaceInvalidState) {
			s.logger.Debug("Namespace '" + namespace.Name + "' invalid state. " + err.Error())
			return nil
		}

		var namespaceNotFound *serviceerror.NamespaceNotFound
		if errors.As(err, &namespaceNotFound) {
			s.logger.Debug("Namespace '" + namespace.Name + "' not found. " + err.Error())
			return nil
		}

		if err != nil {
			return err
		}

		s.logger.Debug("Namespace '" + namespace.Name + "' deleted. Temporary namespace name that is used during reclaim resources step: '" + response.DeletedNamespace + "' ")
	}

	if err != nil {
		return err
	}

	return nil
}

func mapDescribeNamespaceResponse(response *workflowservice.DescribeNamespaceResponse) *core.TemporalNamespaceObservation {
	return &core.TemporalNamespaceObservation{
		Id:                             response.NamespaceInfo.Id,
		Name:                           response.NamespaceInfo.Name,
		Description:                    response.NamespaceInfo.Description,
		OwnerEmail:                     response.NamespaceInfo.OwnerEmail,
		WorkflowExecutionRetentionDays: int(*response.Config.WorkflowExecutionRetentionTtl / day),
		Data:                           response.NamespaceInfo.Data,
		HistoryArchivalState:           response.Config.HistoryArchivalState.String(),
		HistoryArchivalUri:             response.Config.HistoryArchivalUri,
		VisibilityArchivalState:        response.Config.VisibilityArchivalState.String(),
		VisibilityArchivalUri:          response.Config.VisibilityArchivalUri,
		State:                          response.NamespaceInfo.State.String(),
	}
}

func (s *TemporalServiceImpl) ListAllNamespaces(ctx context.Context) ([]*core.TemporalNamespaceObservation, error) {
	// TODO: Pagination (method only used in tests)
	request := &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	}

	responses, err := s.client.WorkflowService().ListNamespaces(ctx, request)
	if err != nil {
		return nil, err
	}

	var namespaces = []*core.TemporalNamespaceObservation{}
	for _, response := range responses.Namespaces {
		namespace := mapDescribeNamespaceResponse(response)
		if namespace.Name != "temporal-system" && namespace.State != "Deleted" {
			namespaces = append(namespaces, namespace)
		}
	}

	return namespaces, nil
}

func (s *TemporalServiceImpl) UpdateNamespaceByName(ctx context.Context, namespace *core.TemporalNamespaceParameters) error {
	updaterequest := &workflowservice.UpdateNamespaceRequest{
		Namespace: namespace.Name,
		UpdateInfo: &ns.UpdateNamespaceInfo{
			Description: namespace.Description,
			OwnerEmail:  namespace.OwnerEmail,
		},
	}

	_, err := s.client.WorkflowService().UpdateNamespace(ctx, updaterequest)

	if err != nil {
		return err
	}

	return nil
}
