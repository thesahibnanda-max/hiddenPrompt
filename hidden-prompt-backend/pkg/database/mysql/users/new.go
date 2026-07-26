package usersRepo

import (
	"context"
	"database/sql"
	"fmt"
	usersInternalMySQL "hidden-prompt-backend/pkg/database/mysql/users/internal"
	"strings"

	_ "embed"

	_ "github.com/go-sql-driver/mysql"
	"vitess.io/vitess/go/vt/sqlparser"
)

//go:embed schema.sql
var schemaSQLString string

func NewRepository(db *sql.DB) (Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

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

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return nil, fmt.Errorf("executing schema statement %q: %w", stmt, err)
		}
	}

	return &repo{
		q: usersInternalMySQL.New(db),
	}, nil
}
