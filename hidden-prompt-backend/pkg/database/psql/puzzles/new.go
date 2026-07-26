package puzzlesRepo

import (
	"context"
	_ "embed"
	"fmt"
	puzzlesInternalPSQL "hidden-prompt-backend/pkg/database/psql/puzzles/internal"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"vitess.io/vitess/go/vt/sqlparser"
)

//go:embed schema.sql
var schemaSQLString string

func NewRepo(db *pgxpool.Pool) (Repository, error) {
	stmts, err := sqlparser.NewTestParser().SplitStatementToPieces(schemaSQLString)
	if err != nil {
		return nil, fmt.Errorf("split schema: %w", err)
	}

	ctx := context.Background()

	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := db.Exec(ctx, stmt); err != nil {
			return nil, fmt.Errorf("executing schema statement %q: %w", stmt, err)
		}
	}

	return &repo{
		q: puzzlesInternalPSQL.New(db),
	}, nil
}
