package slts

import (
	puzzlesRepo "hidden-prompt-backend/pkg/database/psql/puzzles"
	"sync"
)

// Test_AppendPuzzleMessage_ConcurrentWritesDoNotLoseData is the direct
// regression test for the write-race this optimistic-concurrency fix
// closes: AppendPuzzleMessage used to take the caller's already-loaded
// message list and blindly overwrite the row, so two requests reading the
// same puzzle before either wrote back would race - the second write
// silently discarded the first's message instead of erroring. Both
// goroutines here start from the identical snapshot (same messages, same
// updated_at) the way two near-simultaneous guesses on the same puzzle
// would in production. Exactly one must succeed; the other must fail with
// ErrConcurrentModification rather than silently winning and dropping the
// first message.
func (s *sltTestSuite) Test_AppendPuzzleMessage_ConcurrentWritesDoNotLoseData() {
	email := s.freshVerifiedUser()
	id, _, _ := s.freshPuzzle(email)

	snapshot, err := s.dbParams.PuzzlesRepository.GetPuzzleByID(s.ctx, id)
	s.Require().NoError(err)

	const concurrentWriters = 2
	var wg sync.WaitGroup
	results := make([]error, concurrentWriters)

	for i := range concurrentWriters {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, results[i] = s.dbParams.PuzzlesRepository.AppendPuzzleMessage(
				s.ctx, id, snapshot.Messages, snapshot.UpdatedAt, puzzlesRepo.UserRole, "concurrent guess",
			)
		}(i)
	}
	wg.Wait()

	successes, conflicts := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case err == puzzlesRepo.ErrConcurrentModification:
			conflicts++
		default:
			s.Require().NoError(err, "unexpected error type from concurrent AppendPuzzleMessage")
		}
	}
	s.Require().Equal(1, successes, "exactly one concurrent writer starting from the same snapshot must win")
	s.Require().Equal(concurrentWriters-1, conflicts, "every other writer must see ErrConcurrentModification, not a silent lost update")

	final, err := s.dbParams.PuzzlesRepository.GetPuzzleByID(s.ctx, id)
	s.Require().NoError(err)
	s.Require().Len(final.Messages.MessageList, len(snapshot.Messages.MessageList)+1,
		"exactly one message must have been appended - neither lost nor double-applied")
}
