package dao

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

const (
	tLikes     = "post_likes"
	tBookmarks = "post_bookmarks"
)

// IsLiked reports whether the user has liked the post.
func (p *PG) IsLiked(ctx context.Context, user, postID string) (bool, error) {
	n, err := p.db.Model(tLikes).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Count()
	return n > 0, err
}

// IsBookmarked reports whether the user has bookmarked the post.
func (p *PG) IsBookmarked(ctx context.Context, user, postID string) (bool, error) {
	n, err := p.db.Model(tBookmarks).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Count()
	return n > 0, err
}

// ToggleLike flips the user's like on a post and keeps post_stats.like_count in
// sync, atomically. Returns the new liked state.
func (p *PG) ToggleLike(ctx context.Context, user, postID string) (bool, error) {
	liked := false
	err := p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		n, err := tx.Model(tLikes).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Count()
		if err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO post_stats (post_id) VALUES (?) ON CONFLICT (post_id) DO NOTHING", postID); err != nil {
			return err
		}
		if n > 0 {
			if _, err := tx.Model(tLikes).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Delete(); err != nil {
				return err
			}
			if _, err := tx.Model(tStats).Ctx(ctx).Where("post_id", postID).Decrement("like_count", 1); err != nil {
				return err
			}
			liked = false
		} else {
			if _, err := tx.Model(tLikes).Ctx(ctx).Data(g.Map{"user_id": user, "post_id": postID}).Insert(); err != nil {
				return err
			}
			if _, err := tx.Model(tStats).Ctx(ctx).Where("post_id", postID).Increment("like_count", 1); err != nil {
				return err
			}
			liked = true
		}
		return nil
	})
	return liked, err
}

// ToggleBookmark flips the user's bookmark on a post (no aggregate counter).
func (p *PG) ToggleBookmark(ctx context.Context, user, postID string) (bool, error) {
	n, err := p.db.Model(tBookmarks).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Count()
	if err != nil {
		return false, err
	}
	if n > 0 {
		_, err := p.db.Model(tBookmarks).Ctx(ctx).Where("user_id", user).Where("post_id", postID).Delete()
		return false, err
	}
	_, err = p.db.Model(tBookmarks).Ctx(ctx).Data(g.Map{"user_id": user, "post_id": postID}).Insert()
	return true, err
}
