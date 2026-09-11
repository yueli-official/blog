// Package assetreferences owns this consumer's authoritative business usage.
package assetreferences

import (
	"context"
	"database/sql"
	"github.com/yueli-official/asset/referencesync"
)

func Source(origins ...string) func(context.Context, *sql.Tx) ([]referencesync.Snapshot, error) {
	return referencesync.QuerySource([]referencesync.Query{
		{RefType: "post-body", Kind: "markdown", SQL: `SELECT id::text,title,'/manage/'||id::text,content FROM posts WHERE deleted_at IS NULL`},
		{RefType: "post-cover", Kind: "asset", SQL: `SELECT id::text,title,'/manage/'||id::text,cover_asset_id FROM posts WHERE deleted_at IS NULL`},
		{RefType: "series-cover", Kind: "asset", SQL: `SELECT id::text,name,'/series/'||slug,cover_asset_id FROM series`},
		{RefType: "post-seo", Kind: "markdown", SQL: `SELECT p.id::text,p.title,'/manage/'||p.id::text,s.og_image FROM post_seo s JOIN posts p ON p.id=s.post_id WHERE p.deleted_at IS NULL`},
	}, origins...)
}
