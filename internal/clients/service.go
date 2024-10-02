package clients

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"go.temporal.io/sdk/client"
)

type TemporalServiceConfig struct {
	HostPort  string `json:"hostPort"`
	UseTLS    bool   `json:"useTLS"`
	CACertPem string `json:"caCertPem"`
	CertPem   string `json:"certPem"`
	KeyPem    string `json:"keyPem"`
}

type TemporalServiceImpl struct {
	client client.Client
	logger *slog.Logger
}

func NewTemporalService(configData []byte) (*TemporalServiceImpl, error) {
	var conf = TemporalServiceConfig{}
	err := json.Unmarshal(configData, &conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config data")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	logger.Debug("Starting NewTemporalService", slog.String("hostPort", conf.HostPort), slog.Bool("useTLS", conf.UseTLS))

	var dialOptions []grpc.DialOption
	if conf.UseTLS {
		if conf.CACertPem == "" || conf.CertPem == "" || conf.KeyPem == "" {
			return nil, errors.New("TLS is enabled but one or more of the certificates or key are missing")
		}

		logger.Debug("Loading client certificate from strings")
		cert, err := tls.X509KeyPair([]byte(conf.CertPem), []byte(conf.KeyPem))
		if err != nil {
			return nil, errors.Wrap(err, "failed to load client certificate")
		}

		logger.Debug("Loading CA certificate from string")
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(conf.CACertPem)) {
			return nil, errors.New("failed to append CA certificate")
		}

		logger.Debug("Creating TLS credentials")
		creds := credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		})
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
	} else {
		logger.Debug("Using insecure credentials")
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	clientOptions := client.Options{
		HostPort: conf.HostPort,
		Logger:   logger,
		ConnectionOptions: client.ConnectionOptions{
			DialOptions: dialOptions,
		},
	}

	logger.Debug("Dialing Temporal client", slog.String("hostPort", conf.HostPort))
	temporalClient, err := client.Dial(clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial Temporal client")
	}

	logger.Debug("Successfully created Temporal client")
	return &TemporalServiceImpl{
		client: temporalClient,
		logger: logger,
	}, nil
}

func (s *TemporalServiceImpl) Close() {
	s.client.Close()
}

func NewSearchAttributeService(configData []byte) (SearchAttributeService, error) {
	return NewTemporalService(configData)
}

func NewNamespaceService(configData []byte) (NamespaceService, error) {
	return NewTemporalService(configData)
}
