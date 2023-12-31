package clients

import (
	"testing"
)

func createTemporalService(t *testing.T) *TemporalServiceImpl {
	jsonConfig := `{
		"HostPort": "localhost:7222"
	}`

	temporalService := createTemporalServiceWithConfig(t, jsonConfig)

	return temporalService
}

func createTemporalServiceWithConfig(t *testing.T, jsonConfig string) *TemporalServiceImpl {
	service, err := NewTemporalService([]byte(jsonConfig))
	if err != nil {
		t.Fatal(err)
	}
	return service
}
