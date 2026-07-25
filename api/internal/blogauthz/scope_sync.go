package blogauthz

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yueli-official/foundation/go/authorization"
)

func SyncResourceScopes(
	ctx context.Context,
	db *sql.DB,
	runtime authorization.ResourceScopeRegistry,
) error {
	rows, err := db.QueryContext(ctx, `SELECT id::text FROM posts WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list blog posts for authorization scope sync: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan blog post authorization scope: %w", err)
		}
		if err := ensureScope(ctx, runtime, authorization.RegisterScopeCommand{
			ID: PostScopeID(id), Type: ScopePost, ParentID: RootScopeID,
		}); err != nil {
			_ = rows.Close()
			return err
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = db.QueryContext(ctx, `SELECT id::text FROM series ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list blog series for authorization scope sync: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		if err := ensureScope(ctx, runtime, authorization.RegisterScopeCommand{
			ID: SeriesScopeID(id), Type: ScopeSeries, ParentID: RootScopeID,
		}); err != nil {
			_ = rows.Close()
			return err
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = db.QueryContext(ctx, `SELECT id::text, post_id::text FROM comments ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list blog comments for authorization scope sync: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, postID string
		if err := rows.Scan(&id, &postID); err != nil {
			return err
		}
		if err := ensureScope(ctx, runtime, authorization.RegisterScopeCommand{
			ID: CommentScopeID(id), Type: ScopeComment, ParentID: PostScopeID(postID),
		}); err != nil {
			return err
		}
	}
	return rows.Err()
}

func ensureScope(
	ctx context.Context,
	runtime authorization.ResourceScopeRegistry,
	command authorization.RegisterScopeCommand,
) error {
	_, err := runtime.RegisterScope(ctx, command)
	if err == nil || authorization.Is(err, authorization.ErrorConflict) {
		return nil
	}
	return fmt.Errorf("register blog authorization scope %q: %w", command.ID, err)
}
