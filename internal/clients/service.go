package clients

import (
	"encoding/json"
	"os"

	"golang.org/x/exp/slog"

	"go.temporal.io/sdk/client"
)

type TemporalServiceConfig struct {
	HostPort string `json:"hostPort"`
}

type TemporalServiceImpl struct {
	client client.Client
	logger *slog.Logger
}

func NewTemporalService(configData []byte) (NamespaceService, error) {
	var conf = TemporalServiceConfig{}
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

	return &TemporalServiceImpl{
		client: temporalClient,
		logger: logger,
	}, err
}
