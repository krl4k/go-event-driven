package tests

import (
	commonClients "github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ComponentTestSuite struct {
	suite.Suite
	env *TestEnvironment

	// todo use wiremockInstead of this
	gatewayClients *commonClients.Clients
}

func TestComponentTestSuite(t *testing.T) {
	suite.Run(t, new(ComponentTestSuite))
}

func (s *ComponentTestSuite) SetupSuite() {
	env, err := NewTestEnvironment(s.T())
	require.NoError(s.T(), err)

	s.gatewayClients, err = commonClients.NewClients(
		env.GatewayURL,
		nil)
	require.NoError(s.T(), err)
	s.env = env
}

func (s *ComponentTestSuite) TearDownSuite() {
	if s.env != nil {
		s.env.Cleanup()
	}
}

func (s *ComponentTestSuite) SetupTest() {
}

func (s *ComponentTestSuite) TearDownTest() {
}
