package clients

import (
	"testing"
)

func createTemporalService(t *testing.T) *TemporalServiceImpl {
	jsonConfig := `{
		"HostPort": "localhost:7233"
	}`

	temporalService := createTemporalServiceWithConfig(t, jsonConfig)

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
