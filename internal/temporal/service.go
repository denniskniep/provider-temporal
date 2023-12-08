package temporal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"golang.org/x/exp/slog"

	ns "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

type TemporalService interface {
	DescribeNamespaceById(ctx context.Context, id string) (*core.NamespaceObservation, error)
	DescribeNamespaceByName(ctx context.Context, name string) (*core.NamespaceObservation, error)
	CreateNamespace(ctx context.Context, namespace *core.NamespaceParameters) (*core.NamespaceObservation, error)
	DeleteNamespaceById(ctx context.Context, id string) error
	DeleteNamespaceByName(ctx context.Context, name string) error
	DeleteAllNamespaces(ctx context.Context) error
	ListAllNamespaces(ctx context.Context) ([]*core.NamespaceObservation, error)
	UpdateNamespaceById(ctx context.Context, id string, namespace *core.NamespaceParameters) (*core.NamespaceObservation, error)
}

type TemporalServiceImpl struct {
	client client.Client
}

func NewTemporalService(configData []byte) (TemporalService, error) {
	var conf = config{}
	err := json.Unmarshal(configData, &conf)
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	clientOptions := client.Options{
		HostPort: conf.HostPort,
		Logger:   logger,
	}

	temporalClient, err := client.Dial(clientOptions)
	if err != nil {
		return nil, err
	}

	return &TemporalServiceImpl{client: temporalClient}, err
}

func (s *TemporalServiceImpl) CreateNamespace(ctx context.Context, namespace *core.NamespaceParameters) (*core.NamespaceObservation, error) {
	var defaultDuration = 30 * 24 * time.Hour

	createrequest := &workflowservice.RegisterNamespaceRequest{
		Namespace:                        namespace.Name,
		Description:                      namespace.Description,
		OwnerEmail:                       namespace.OwnerEmail,
		WorkflowExecutionRetentionPeriod: &defaultDuration,
	}

	_, err := s.client.WorkflowService().RegisterNamespace(ctx, createrequest)

	createdSuccessfully := err == nil
	var errType *serviceerror.NamespaceAlreadyExists
	alreadyExists := errors.As(err, &errType)

	if createdSuccessfully || alreadyExists {
		return s.DescribeNamespaceByName(ctx, namespace.Name)
	}

	return nil, err
}

func (s *TemporalServiceImpl) DeleteNamespaceById(ctx context.Context, id string) error {
	namespace, err := s.DescribeNamespaceById(ctx, id)

	if err != nil {
		return err
	}

	if namespace == nil {
		return nil
	}

	return s.DeleteNamespaceByName(ctx, namespace.Name)
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

func (s *TemporalServiceImpl) DeleteNamespaceByName(ctx context.Context, name string) error {
	deleterequest := &operatorservice.DeleteNamespaceRequest{
		Namespace: name,
	}

	_, err := s.client.OperatorService().DeleteNamespace(ctx, deleterequest)

	if err != nil {
		return err
	}

	return nil
}

func (s *TemporalServiceImpl) DescribeNamespaceById(ctx context.Context, id string) (*core.NamespaceObservation, error) {
	request := &workflowservice.DescribeNamespaceRequest{
		Id: id,
	}

	response, err := s.client.WorkflowService().DescribeNamespace(ctx, request)
	if err != nil {
		return nil, err
	}

	// Does not exist
	if response == nil {
		return nil, nil
	}

	return mapDescribeNamespaceResponse(response), nil
}

func mapDescribeNamespaceResponse(response *workflowservice.DescribeNamespaceResponse) *core.NamespaceObservation {
	return &core.NamespaceObservation{
		Id:          response.NamespaceInfo.Id,
		Name:        response.NamespaceInfo.Name,
		Description: response.NamespaceInfo.Description,
		OwnerEmail:  response.NamespaceInfo.OwnerEmail,
	}
}

func (s *TemporalServiceImpl) DescribeNamespaceByName(ctx context.Context, name string) (*core.NamespaceObservation, error) {

	request := &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	}

	responses, err := s.client.WorkflowService().ListNamespaces(ctx, request)
	if err != nil {
		return nil, err
	}

	for _, response := range responses.Namespaces {
		if response.NamespaceInfo.Name == name {
			return mapDescribeNamespaceResponse(response), nil
		}
	}

	// Does not exist
	return nil, nil
}

func (s *TemporalServiceImpl) ListAllNamespaces(ctx context.Context) ([]*core.NamespaceObservation, error) {

	request := &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	}

	responses, err := s.client.WorkflowService().ListNamespaces(ctx, request)
	if err != nil {
		return nil, err
	}

	var namespaces = []*core.NamespaceObservation{}
	for _, response := range responses.Namespaces {
		namespace := mapDescribeNamespaceResponse(response)
		if namespace.Name != "temporal-system" {
			namespaces = append(namespaces, namespace)
		}
	}

	return namespaces, nil
}

func (s *TemporalServiceImpl) UpdateNamespaceById(ctx context.Context, id string, namespace *core.NamespaceParameters) (*core.NamespaceObservation, error) {

	found, err := s.DescribeNamespaceById(ctx, id)

	if err != nil {
		return nil, err
	}

	if namespace == nil {
		return nil, nil
	}

	updaterequest := &workflowservice.UpdateNamespaceRequest{
		Namespace: found.Name,
		UpdateInfo: &ns.UpdateNamespaceInfo{
			Description: namespace.Description,
			OwnerEmail:  namespace.OwnerEmail,
		},
	}

	_, err = s.client.WorkflowService().UpdateNamespace(ctx, updaterequest)

	if err != nil {
		return nil, err
	}

	afterUpdate, err := s.DescribeNamespaceByName(ctx, namespace.Name)
	if err != nil {
		return nil, err
	}

	return afterUpdate, nil
}
