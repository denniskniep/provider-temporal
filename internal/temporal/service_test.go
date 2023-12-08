package temporal

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"golang.org/x/net/context"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

func createTemporalService(t *testing.T) TemporalService {

	jsonConfig := `{
		"HostPort": "temporal.k8s.local:7233"
	}`

	temporalService := createTemporalServiceWithConfig(t, jsonConfig)
	err := temporalService.DeleteAllNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertNamespacesCount(t, temporalService, 0)
	return temporalService
}

func createTemporalServiceWithConfig(t *testing.T, jsonConfig string) TemporalService {
	service, err := NewTemporalService([]byte(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCreate(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace := &core.NamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}

	created, err := temporalService.CreateNamespace(context.Background(), testNamespace)

	if err != nil {
		t.Fatal(err)
	}

	assertNamespaceAreEqual(t, created, testNamespace)
	assertNamespacesCount(t, temporalService, 1)
}

func TestCreateUpdate(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.NamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}
	created1, err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)

	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	testNamespace2 := &core.NamespaceParameters{
		Name:        "Test2",
		Description: "Desc2",
		OwnerEmail:  "Test2@mail.local",
	}
	created2, err2 := temporalService.CreateNamespace(context.Background(), testNamespace2)

	if err2 != nil {
		t.Fatal(err2)
	}

	assertNamespaceAreEqual(t, created1, testNamespace1)
	assertNamespaceAreEqual(t, created2, testNamespace2)
	assertNamespacesCount(t, temporalService, 2)

	testNamespaceUpdate := &core.NamespaceParameters{
		Name:        "Test2",
		Description: "Updated2",
		OwnerEmail:  "Updated2@mail.local",
	}
	updated, err := temporalService.UpdateNamespaceById(context.Background(), created2.Id, testNamespaceUpdate)

	if err != nil {
		t.Fatal(err)
	}

	assertNamespaceAreEqual(t, created1, testNamespace1)
	assertNamespaceAreEqual(t, updated, testNamespaceUpdate)
	assertNamespacesCount(t, temporalService, 2)
}

func TestCreateDeleteByName(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.NamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}
	created1, err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)

	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	temporalService.DeleteNamespaceByName(context.Background(), created1.Name)

	assertNamespacesCount(t, temporalService, 0)
}

func TestCreateDeleteById(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.NamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}
	created1, err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)

	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	temporalService.DeleteNamespaceById(context.Background(), created1.Id)

	assertNamespacesCount(t, temporalService, 0)
}

func assertNamespaceAreEqual(t *testing.T, actual *core.NamespaceObservation, expected *core.NamespaceParameters) {
	mappedActual, err := mapToNamespaceParameters(actual)
	if err != nil {
		t.Fatal(err)
	}
	diff := cmp.Diff(mappedActual, expected)
	if diff != "" {
		t.Fatal(diff)
	}
}

func assertNamespacesCount(t *testing.T, temporalService TemporalService, expectedCount int) {
	namespaces, err := temporalService.ListAllNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != expectedCount {
		t.Fatal("Expected Namespace Count is " + strconv.Itoa(expectedCount) + ", but was " + strconv.Itoa(len(namespaces)))
	}
}

func skipIfIsShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
}

func mapToNamespaceParameters(ns *core.NamespaceObservation) (*core.NamespaceParameters, error) {
	nsJson, err := json.Marshal(ns)
	if err != nil {
		return nil, err
	}

	var nsParam = core.NamespaceParameters{}
	err = json.Unmarshal(nsJson, &nsParam)
	if err != nil {
		return nil, err
	}

	return &nsParam, nil
}
