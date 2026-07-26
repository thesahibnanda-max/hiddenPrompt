package slts

import (
	"context"
	"hidden-prompt-backend/pkg/core/deps"
	"hidden-prompt-backend/pkg/database"
	env "hidden-prompt-backend/pkg/environment"
	"hidden-prompt-backend/pkg/service/puzzle"
	"hidden-prompt-backend/pkg/service/user"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
)

type sltTestSuite struct {
	suite.Suite

	ctx           context.Context
	cancel        context.CancelFunc
	app           *fx.App
	dbParams      *database.DatabaseParams
	puzzleService puzzle.Service
	userService   user.Service
	// createdEmails tracks every user freshVerifiedUser creates during the
	// run, so TearDownSuite can delete them (and their puzzles) afterward -
	// without this, every run permanently adds real users/puzzles to
	// whatever database SetupSuite points at.
	createdEmails []string
}

func (s *sltTestSuite) SetupSuite() {
	env.SetDotEnv("../../")

	var databasePopulate *database.DatabaseParams
	var puzzleService puzzle.Service
	var userService user.Service
	s.app = deps.NewApp(fx.Populate(&databasePopulate), fx.Populate(&puzzleService), fx.Populate(&userService))
	s.puzzleService = puzzleService
	s.userService = userService
	s.dbParams = databasePopulate
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.Require().NoError(s.app.Start(ctx))
}

func (s *sltTestSuite) TearDownSuite() {
	defer s.cancel()

	// Best-effort: logged rather than s.Require().NoError, since a cleanup
	// hiccup here shouldn't mask or fail otherwise-passing test results.
	for _, email := range s.createdEmails {
		if err := s.dbParams.PuzzlesRepository.DeletePuzzlesByEmail(s.ctx, email); err != nil {
			s.T().Logf("teardown: failed to delete puzzles for %s: %v", email, err)
		}
		if err := s.dbParams.UsersRepository.DeleteUserByEmail(s.ctx, email); err != nil {
			s.T().Logf("teardown: failed to delete user %s: %v", email, err)
		}
	}

	s.Require().NoError(s.app.Stop(s.ctx))
}

func Test_Suite(t *testing.T) {
	suite.Run(t, new(sltTestSuite))
}
