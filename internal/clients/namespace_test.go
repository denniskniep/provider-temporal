package clients

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"golang.org/x/net/context"

	core "github.com/denniskniep/provider-temporal/apis/core/v1alpha1"
)

func createTemporalNamespaceService(t *testing.T) *TemporalServiceImpl {
	temporalService := createTemporalService(t)

	err := temporalService.DeleteAllNamespaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertNamespacesCount(t, temporalService, 0)
	return temporalService
}

func createDefaultNamespaceParameters() *core.TemporalNamespaceParameters {
	desc := "Desc1"
	mail := "Test1@mail.local"
	return &core.TemporalNamespaceParameters{
		Name:                           "Test1",
		Description:                    &desc,
		OwnerEmail:                     &mail,
		WorkflowExecutionRetentionDays: 30,
		HistoryArchivalState:           "Disabled",
		VisibilityArchivalState:        "Disabled",
	}
}

func createDefaultNamespaceParametersWithName(name string) *core.TemporalNamespaceParameters {
	ns := createDefaultNamespaceParameters()
	ns.Name = name
	return ns
}

func TestDeleteTwice(t *testing.T) {
	skipIfIsShort(t)

	temporalService := createTemporalNamespaceService(t)
	testNamespace := createDefaultNamespaceParameters()

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

	temporalService := createTemporalNamespaceService(t)

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

	temporalService := createTemporalNamespaceService(t)
	testNamespace := createDefaultNamespaceParameters()

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

	temporalService := createTemporalNamespaceService(t)
	testNamespace1 := createDefaultNamespaceParameters()
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

	desc2 := "Desc2"
	mail2 := "Test2@mail.local"
	testNamespace2 := createDefaultNamespaceParametersWithName("Test2")
	testNamespace2.Description = &desc2
	testNamespace2.OwnerEmail = &mail2

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

	updatedDesc := "Updated2"
	updatedMail := "Updated2@mail.local"
	testNamespaceUpdate := createDefaultNamespaceParametersWithName("Test2")
	testNamespace2.Description = &updatedDesc
	testNamespace2.OwnerEmail = &updatedMail

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

	temporalService := createTemporalNamespaceService(t)
	testNamespace1 := createDefaultNamespaceParameters()
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

	temporalService := createTemporalNamespaceService(t)
	testNamespace1 := createDefaultNamespaceParameters()

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

func assertNamespaceAreEqual(t *testing.T, temporalService NamespaceService, actual *core.TemporalNamespaceObservation, expected *core.TemporalNamespaceParameters) {
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
