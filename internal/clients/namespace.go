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
	DeleteNamespaceByName(ctx context.Context, name string) (*string, error)

	MapToNamespaceCompare(namespace interface{}) (*NamespaceCompare, error)
}

type NamespaceCompare struct {
	Name                           string             `json:"name"`
	Description                    *string            `json:"description,omitempty"`
	OwnerEmail                     *string            `json:"ownerEmail,omitempty"`
	WorkflowExecutionRetentionDays int                `json:"workflowExecutionRetentionDays,omitempty"`
	Data                           *map[string]string `json:"data,omitempty"`
	HistoryArchivalState           string             `json:"historyArchivalState,omitempty"`
	HistoryArchivalUri             *string            `json:"historyArchivalUri,omitempty"`
	VisibilityArchivalState        string             `json:"visibilityArchivalState,omitempty"`
	VisibilityArchivalUri          *string            `json:"visibilityArchivalUri,omitempty"`
}

func (s *TemporalServiceImpl) MapToNamespaceCompare(namespace interface{}) (*NamespaceCompare, error) {
	namespaceJson, err := json.Marshal(namespace)
	if err != nil {
		return nil, err
	}

	var namespaceCompare = NamespaceCompare{}
	err = json.Unmarshal(namespaceJson, &namespaceCompare)
	if err != nil {
		return nil, err
	}

	return &namespaceCompare, nil
}

func (s *TemporalServiceImpl) CreateNamespace(ctx context.Context, namespace *core.TemporalNamespaceParameters) error {
	retentionDuration := time.Duration(namespace.WorkflowExecutionRetentionDays) * day

	var data map[string]string
	if namespace.Data != nil {
		data = *namespace.Data
	}

	createrequest := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace.Name,
		Description:                      resolvePtrOrDefault(namespace.Description),
		OwnerEmail:                       resolvePtrOrDefault(namespace.OwnerEmail),
		WorkflowExecutionRetentionPeriod: &retentionDuration,
		Data:                             data,
		HistoryArchivalState:             enums.ArchivalState(enums.ArchivalState_value[namespace.HistoryArchivalState]),
		HistoryArchivalUri:               resolvePtrOrDefault(namespace.HistoryArchivalUri),
		VisibilityArchivalState:          enums.ArchivalState(enums.ArchivalState_value[namespace.VisibilityArchivalState]),
		VisibilityArchivalUri:            resolvePtrOrDefault(namespace.VisibilityArchivalUri),
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

func (s *TemporalServiceImpl) DeleteAllNamespaces(ctx context.Context) ([]*string, error) {
	namespaces, err := s.ListAllNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	deletedNamespaces := make([]*string, 0, len(namespaces))
	for _, namespace := range namespaces {
		deletedNamespace, err := s.DeleteNamespaceByName(ctx, namespace.Name)
		if err != nil {
			return deletedNamespaces, err
		}
		deletedNamespaces = append(deletedNamespaces, deletedNamespace)
	}

	return deletedNamespaces, nil
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

func (s *TemporalServiceImpl) DeleteNamespaceByName(ctx context.Context, name string) (*string, error) {
	deleterequest := &operatorservice.DeleteNamespaceRequest{
		Namespace: name,
	}

	namespace, err := s.DescribeNamespaceByName(ctx, name)
	if namespace != nil {
		response, err := s.client.OperatorService().DeleteNamespace(ctx, deleterequest)

		var namespaceInvalidState *serviceerror.NamespaceInvalidState
		if errors.As(err, &namespaceInvalidState) {
			s.logger.Debug("Namespace '" + namespace.Name + "' invalid state! " + err.Error())
			return &namespace.Name, nil
		}

		var namespaceNotFound *serviceerror.NamespaceNotFound
		if errors.As(err, &namespaceNotFound) {
			s.logger.Debug("Namespace '" + namespace.Name + "' not found! " + err.Error())
			return &namespace.Name, nil
		}

		if err != nil {
			return &namespace.Name, err
		}

		s.logger.Debug("Namespace '" + namespace.Name + "' deleted. Temporary namespace name that is used during reclaim resources step: '" + response.DeletedNamespace + "' ")
		return &namespace.Name, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func mapDescribeNamespaceResponse(response *workflowservice.DescribeNamespaceResponse) *core.TemporalNamespaceObservation {
	var data *map[string]string = nil
	if len(response.NamespaceInfo.Data) > 0 {
		data = &response.NamespaceInfo.Data
	}

	return &core.TemporalNamespaceObservation{
		Id:                             response.NamespaceInfo.Id,
		Name:                           response.NamespaceInfo.Name,
		Description:                    createPtrOrNilIfDefault(response.NamespaceInfo.Description),
		OwnerEmail:                     createPtrOrNilIfDefault(response.NamespaceInfo.OwnerEmail),
		WorkflowExecutionRetentionDays: int(*response.Config.WorkflowExecutionRetentionTtl / day),
		Data:                           data,
		HistoryArchivalState:           response.Config.HistoryArchivalState.String(),
		HistoryArchivalUri:             createPtrOrNilIfDefault(response.Config.HistoryArchivalUri),
		VisibilityArchivalState:        response.Config.VisibilityArchivalState.String(),
		VisibilityArchivalUri:          createPtrOrNilIfDefault(response.Config.VisibilityArchivalUri),
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

	retentionTtl := time.Duration(namespace.WorkflowExecutionRetentionDays * int(day))

	var data map[string]string
	if namespace.Data != nil {
		data = *namespace.Data
	}

	updaterequest := &workflowservice.UpdateNamespaceRequest{
		Namespace: namespace.Name,
		UpdateInfo: &ns.UpdateNamespaceInfo{
			Description: resolvePtrOrDefault(namespace.Description),
			OwnerEmail:  resolvePtrOrDefault(namespace.OwnerEmail),
			Data:        data,
		},
		Config: &ns.NamespaceConfig{
			HistoryArchivalState:          enums.ArchivalState(enums.ArchivalState_value[namespace.HistoryArchivalState]),
			HistoryArchivalUri:            resolvePtrOrDefault(namespace.HistoryArchivalUri),
			VisibilityArchivalState:       enums.ArchivalState(enums.ArchivalState_value[namespace.VisibilityArchivalState]),
			VisibilityArchivalUri:         resolvePtrOrDefault(namespace.VisibilityArchivalUri),
			WorkflowExecutionRetentionTtl: &retentionTtl,
		},
	}

	_, err := s.client.WorkflowService().UpdateNamespace(ctx, updaterequest)

	if err != nil {
		return err
	}

	return nil
}

func resolvePtrOrDefault(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func createPtrOrNilIfDefault(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
