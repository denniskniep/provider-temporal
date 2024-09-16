package clients

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/context"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

func createSearchAttributeService(t *testing.T) *TemporalServiceImpl {
	temporalService := createTemporalService(t)
	return temporalService
}

func createSearchAttributeServiceTLS(t *testing.T) *TemporalServiceImpl {
	temporalService := createTemporalServiceTLS(t)
	return temporalService
}

func createSearchAttributeParameters(namespace string, attrName string, attrType string) *core.SearchAttributeParameters {
	return &core.SearchAttributeParameters{
		Name:                  attrName,
		Type:                  attrType,
		TemporalNamespaceName: &namespace,
	}
}

func TestCreateSearchAttribute(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createSearchAttributeService(t)
	testNamespace := createDefaultNamespaceParametersWithName("Test010")

	err := temporalService.CreateNamespace(context.Background(), testNamespace)
	if err != nil {
		t.Fatal(err)
	}

	testAttr := createSearchAttributeParameters(testNamespace.Name, "test1", "Keyword")
	temporalService.CreateSearchAttribute(context.Background(), testAttr)

	foundSearchAttr, err := temporalService.DescribeSearchAttributeByName(context.Background(), testNamespace.Name, testAttr.Name)
	if err != nil {
		t.Fatal(err)
	}

	assertSearchAttributesAreEqual(t, temporalService, foundSearchAttr, testAttr)
	assertSearchAttributeCount(t, temporalService, testNamespace.Name, 1)

	temporalService.DeleteSearchAttributeByName(context.Background(), testNamespace.Name, testAttr.Name)
	assertSearchAttributeCount(t, temporalService, testNamespace.Name, 0)
}

func TestCreateSearchAttributeTLS(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createSearchAttributeServiceTLS(t)
	testNamespace := createDefaultNamespaceParametersWithName("Test010")

	err := temporalService.CreateNamespace(context.Background(), testNamespace)
	if err != nil {
		t.Fatal(err)
	}

	testAttr := createSearchAttributeParameters(testNamespace.Name, "test1TLS", "Keyword")
	temporalService.CreateSearchAttribute(context.Background(), testAttr)

	foundSearchAttr, err := temporalService.DescribeSearchAttributeByName(context.Background(), testNamespace.Name, testAttr.Name)
	if err != nil {
		t.Fatal(err)
	}

	assertSearchAttributesAreEqual(t, temporalService, foundSearchAttr, testAttr)
	assertSearchAttributeCount(t, temporalService, testNamespace.Name, 1)

	temporalService.DeleteSearchAttributeByName(context.Background(), testNamespace.Name, testAttr.Name)
	assertSearchAttributeCount(t, temporalService, testNamespace.Name, 0)
}

func assertSearchAttributesAreEqual(t *testing.T, temporalService SearchAttributeService, actual *core.SearchAttributeObservation, expected *core.SearchAttributeParameters) {
	mappedActual, err := temporalService.MapToSearchAttributeCompare(actual)
	if err != nil {
		t.Fatal(err)
	}

	mappedExpected, err := temporalService.MapToSearchAttributeCompare(expected)
	if err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(mappedActual, mappedExpected)
	if diff != "" {
		t.Fatal(diff)
	}
}

func assertSearchAttributeCount(t *testing.T, temporalService *TemporalServiceImpl, namespace string, expectedCount int) {
	t.Helper()
	searchAttributes, err := temporalService.ListSearchAttributesByNamespace(context.Background(), namespace)
	if err != nil {
		t.Fatal(err)
	}
	if len(searchAttributes) != expectedCount {
		searchAttributesAsJson, err := json.Marshal(searchAttributes)
		if err != nil {
			t.Error(err)
		}

		t.Fatal("Expected SearchAttribute Count is " + strconv.Itoa(expectedCount) + ", but was " + strconv.Itoa(len(searchAttributes)) + "\n" + string(searchAttributesAsJson))
	}
}
