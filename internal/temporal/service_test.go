package temporal

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"golang.org/x/net/context"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

func createTemporalService(t *testing.T) *TemporalServiceImpl {
	jsonConfig := `{
		"HostPort": "temporal.k8s.localhost:7233"
	}`

	temporalService := createTemporalServiceWithConfig(t, jsonConfig)

	err := temporalService.DeleteAllNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertNamespacesCount(t, temporalService, 0)
	return temporalService
}

func createTemporalServiceWithConfig(t *testing.T, jsonConfig string) *TemporalServiceImpl {
	service, err := NewTemporalService([]byte(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}

	impl, ok := service.(*TemporalServiceImpl)
	if !ok {
		t.Fatal("Not of type TemporalServiceImpl")
	}
	return impl
}

func TestDeleteTwice(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace := &core.TemporalNamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}

	err := temporalService.CreateNamespace(context.Background(), testNamespace)
	if err != nil {
		t.Fatal(err)
	}

	err = temporalService.DeleteNamespaceByName(context.Background(), testNamespace.Name)
	if err != nil {
		t.Fatal(err)
	}

	err = temporalService.DeleteNamespaceByName(context.Background(), testNamespace.Name)
	if err != nil {
		t.Fatal(err)
	}

	assertNamespacesCount(t, temporalService, 0)
}

func TestDescribeNotExistingNamespace(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	namespace, err := temporalService.DescribeNamespaceByName(context.Background(), "DoesNotExist")
	if err != nil {
		t.Fatal(err)
	}

	if namespace != nil {
		t.Fatal("Namespace should not exist")
	}
}

func TestCreate(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace := &core.TemporalNamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}

	err := temporalService.CreateNamespace(context.Background(), testNamespace)
	if err != nil {
		t.Fatal(err)
	}

	created, err := temporalService.DescribeNamespaceByName(context.Background(), testNamespace.Name)
	if err != nil {
		t.Fatal(err)
	}

	assertNamespaceAreEqual(t, temporalService, created, testNamespace)
	assertNamespacesCount(t, temporalService, 1)
}

func TestCreateUpdate(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.TemporalNamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}
	err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)
	if err1 != nil {
		t.Fatal(err1)
	}

	created1, err1 := temporalService.DescribeNamespaceByName(context.Background(), testNamespace1.Name)
	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, temporalService, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	testNamespace2 := &core.TemporalNamespaceParameters{
		Name:        "Test2",
		Description: "Desc2",
		OwnerEmail:  "Test2@mail.local",
	}
	err2 := temporalService.CreateNamespace(context.Background(), testNamespace2)
	if err2 != nil {
		t.Fatal(err2)
	}

	created2, err2 := temporalService.DescribeNamespaceByName(context.Background(), testNamespace2.Name)
	if err2 != nil {
		t.Fatal(err2)
	}

	assertNamespaceAreEqual(t, temporalService, created1, testNamespace1)
	assertNamespaceAreEqual(t, temporalService, created2, testNamespace2)
	assertNamespacesCount(t, temporalService, 2)

	testNamespaceUpdate := &core.TemporalNamespaceParameters{
		Name:        "Test2",
		Description: "Updated2",
		OwnerEmail:  "Updated2@mail.local",
	}
	err := temporalService.UpdateNamespaceByName(context.Background(), testNamespaceUpdate)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := temporalService.DescribeNamespaceByName(context.Background(), testNamespaceUpdate.Name)
	if err != nil {
		t.Fatal(err)
	}

	assertNamespaceAreEqual(t, temporalService, created1, testNamespace1)
	assertNamespaceAreEqual(t, temporalService, updated, testNamespaceUpdate)
	assertNamespacesCount(t, temporalService, 2)
}

func TestCreateDeleteByName(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.TemporalNamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}
	err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)
	if err1 != nil {
		t.Fatal(err1)
	}

	created1, err1 := temporalService.DescribeNamespaceByName(context.Background(), testNamespace1.Name)
	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, temporalService, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	temporalService.DeleteNamespaceByName(context.Background(), created1.Name)

	assertNamespacesCount(t, temporalService, 0)
}

func TestCreateDeleteById(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalService(t)

	testNamespace1 := &core.TemporalNamespaceParameters{
		Name:        "Test1",
		Description: "Desc1",
		OwnerEmail:  "Test1@mail.local",
	}

	err1 := temporalService.CreateNamespace(context.Background(), testNamespace1)
	if err1 != nil {
		t.Fatal(err1)
	}

	created1, err1 := temporalService.DescribeNamespaceByName(context.Background(), testNamespace1.Name)
	if err1 != nil {
		t.Fatal(err1)
	}

	assertNamespaceAreEqual(t, temporalService, created1, testNamespace1)
	assertNamespacesCount(t, temporalService, 1)

	temporalService.DeleteNamespaceByName(context.Background(), created1.Name)

	assertNamespacesCount(t, temporalService, 0)
}

func assertNamespaceAreEqual(t *testing.T, temporalService TemporalService, actual *core.TemporalNamespaceObservation, expected *core.TemporalNamespaceParameters) {
	mappedActual, err := temporalService.MapObservationToNamespaceParameters(actual)
	if err != nil {
		t.Fatal(err)
	}
	diff := cmp.Diff(mappedActual, expected)
	if diff != "" {
		t.Fatal(diff)
	}
}

func assertNamespacesCount(t *testing.T, temporalService *TemporalServiceImpl, expectedCount int) {
	namespaces, err := temporalService.ListAllNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != expectedCount {
		namespacesAsJson, err := json.Marshal(namespaces)
		t.Error(err)
		t.Fatal("Expected Namespace Count is " + strconv.Itoa(expectedCount) + ", but was " + strconv.Itoa(len(namespaces)) + "\n" + string(namespacesAsJson))
	}
}

func skipIfIsShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
}
