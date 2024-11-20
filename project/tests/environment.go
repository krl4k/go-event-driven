package tests

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/http"
	"testing"
	"time"
)

type TestEnvironment struct {
	Config     TestConfig
	HTTPClient *http.Client

	RedisUrl       string
	redisContainer testcontainers.Container

	ServiceURL       string
	serviceContainer testcontainers.Container

	GatewayURL       string
	gatewayContainer testcontainers.Container

	cleanup []func() error
}

func NewTestEnvironment(t *testing.T) (*TestEnvironment, error) {
	config := LoadTestConfig()

	env := &TestEnvironment{
		Config: config,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if err := env.setupRedis(t); err != nil {
		return nil, fmt.Errorf("redis setup failed: %v", err)
	}

	if err := env.setupGateway(t); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("wiremock setup failed: %v", err)
	}

	if err := env.setupService(t); err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("service setup failed: %v", err)
	}

	return env, nil
}

func (env *TestEnvironment) setupRedis(t *testing.T) error {
	if env.Config.UseLocalEnv {
		if env.Config.RedisURL == "" {
			return fmt.Errorf("TEST_REDIS_URL is required when USE_LOCAL_REDIS=true")
		}
		env.RedisUrl = env.Config.RedisURL
		return nil
	}

	container, err := startRedisContainer()
	if err != nil {
		return err
	}

	env.redisContainer = container
	env.cleanup = append(env.cleanup, func() error {
		return container.Terminate(context.Background())
	})

	return nil
}

func (env *TestEnvironment) setupService(t *testing.T) error {
	if env.Config.UseLocalEnv {
		if env.Config.ServiceURL == "" {
			return fmt.Errorf("TEST_SERVICE_URL is required when USE_LOCAL_SERVICE=true")
		}

		env.ServiceURL = env.Config.ServiceURL
		return nil
	}

	panic("this code is not implemented yet")
	// todo run service inside container via testcontainers
	return nil
}

func (env *TestEnvironment) setupGateway(t *testing.T) error {
	if env.Config.UseLocalEnv {
		if env.Config.GatewayURL == "" {
			return fmt.Errorf("TEST_GATEWAY_URL is required when USE_LOCAL_GATEWAY=true")
		}

		env.GatewayURL = env.Config.GatewayURL
		fmt.Println("Local Gateway URL: ", env.GatewayURL)
		return nil
	}

	container, err := startGatewayContainer()
	require.NoError(t, err)

	env.gatewayContainer = container
	env.cleanup = append(env.cleanup, func() error {
		return container.Terminate(context.Background())
	})

	port, err := container.MappedPort(context.Background(), "8080/tcp")
	require.NoError(t, err)

	env.GatewayURL = "http://127.0.0.1:" + port.Port()
	fmt.Println("Gateway URL: ", env.GatewayURL)
	return nil
}

func (env *TestEnvironment) Cleanup() {
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		if err := env.cleanup[i](); err != nil {
			fmt.Printf("Cleanup error: %v\n", err)
		}
	}
}

func startRedisContainer() (testcontainers.Container, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:6-2-alpine",
		ExposedPorts: []string{"63799/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

// startServiceContainer run service inside container
//func startServiceContainer(redisAddr, gatewayAddr string) (testcontainers.Container, error) {
//	ctx := context.Background()
//	req := testcontainers.ContainerRequest{
//		Image:        "your-service-image:latest",
//		ExposedPorts: []string{"8090/tcp"},
//		Env: map[string]string{
//			"REDIS_URL":    redisAddr,
//			"GATEWAY_ADDR": gatewayAddr,
//		},
//		WaitingFor: wait.ForHTTP("/health").WithPort("8090/tcp"),
//	}
//
//	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
//		ContainerRequest: req,
//		Started:          true,
//	})
//}

func startGatewayContainer() (testcontainers.Container, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/threedotslabs/event-driven-gateway:latest",
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"SOLUTION_BASE_URL": "http://host.docker.internal:8080/",
		},
		HostConfigModifier: func(config *container.HostConfig) {
			config.PortBindings = nat.PortMap{
				"8080/tcp": []nat.PortBinding{{}},
			}
			config.ExtraHosts = []string{"host.docker.internal:host-gateway"}
		},
		WaitingFor: wait.ForLog("Gateway is starting"),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}
