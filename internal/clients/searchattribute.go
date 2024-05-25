package clients

import (
	"context"
	"encoding/json"

	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

type SearchAttributeService interface {
	DescribeSearchAttributeByName(ctx context.Context, namespace string, name string) (*core.SearchAttributeObservation, error)

	CreateSearchAttribute(ctx context.Context, searchAttribute *core.SearchAttributeParameters) error
	DeleteSearchAttributeByName(ctx context.Context, namespace string, name string) error

	MapToSearchAttributeCompare(searchAttribute interface{}) (*SearchAttributeCompare, error)

	Close()
}

type SearchAttributeCompare struct {
	Name                  string  `json:"name"`
	Type                  string  `json:"type"`
	TemporalNamespaceName *string `json:"temporalNamespaceName,omitempty"`
}

func (s *TemporalServiceImpl) MapToSearchAttributeCompare(searchAttribute interface{}) (*SearchAttributeCompare, error) {
	searchAttributeJson, err := json.Marshal(searchAttribute)
	if err != nil {
		return nil, err
	}

	var searchAttributeCompare = SearchAttributeCompare{}
	err = json.Unmarshal(searchAttributeJson, &searchAttributeCompare)
	if err != nil {
		return nil, err
	}

	return &searchAttributeCompare, nil
}

func (s *TemporalServiceImpl) CreateSearchAttribute(ctx context.Context, searchAttribute *core.SearchAttributeParameters) error {

	searchAttributeMap := make(map[string]enums.IndexedValueType)
	searchAttributeMap[searchAttribute.Name] = enums.IndexedValueType(enums.IndexedValueType_value[searchAttribute.Type])

	createrequest := &operatorservice.AddSearchAttributesRequest{
		Namespace:        *searchAttribute.TemporalNamespaceName,
		SearchAttributes: searchAttributeMap,
	}
	_, err := s.client.OperatorService().AddSearchAttributes(ctx, createrequest)
	if err != nil {
		return err
	}

	return nil
}

func (s *TemporalServiceImpl) DescribeSearchAttributeByName(ctx context.Context, namespace string, name string) (*core.SearchAttributeObservation, error) {
	response, err := s.ListSearchAttributesByNamespace(ctx, namespace)

	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, nil
	}

	for _, customAttribute := range response {
		if customAttribute.Name == name {
			return customAttribute, nil
		}
	}
	return nil, nil
}

func (s *TemporalServiceImpl) ListSearchAttributesByNamespace(ctx context.Context, namespace string) ([]*core.SearchAttributeObservation, error) {
	request := &operatorservice.ListSearchAttributesRequest{
		Namespace: namespace,
	}

	response, err := s.client.OperatorService().ListSearchAttributes(ctx, request)
	if err != nil {
		return nil, err
	}

	var customAttributes = make([]*core.SearchAttributeObservation, 0, len(response.CustomAttributes))

	if response == nil {
		return customAttributes, nil
	}

	for attrName, attrType := range response.CustomAttributes {
		customAttribute := core.SearchAttributeObservation{
			Name:                  attrName,
			Type:                  attrType.String(),
			TemporalNamespaceName: namespace,
		}

		customAttributes = append(customAttributes, &customAttribute)
	}

	return customAttributes, nil
}

func (s *TemporalServiceImpl) DeleteSearchAttributeByName(ctx context.Context, namespace string, name string) error {
	searchAttributeNames := []string{name}

	deleterequest := &operatorservice.RemoveSearchAttributesRequest{
		Namespace:        namespace,
		SearchAttributes: searchAttributeNames,
	}

	_, err := s.client.OperatorService().RemoveSearchAttributes(ctx, deleterequest)
	if err != nil {
		return err
	}

	return nil
}
