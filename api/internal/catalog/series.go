package catalog

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"github.com/yueli-official/blog/api/internal/blogerr"
	"github.com/yueli-official/blog/api/internal/model"
	"github.com/yueli-official/foundation/go/identifier"
)

// CreateSeries makes a new series owned by the author.
func (s *Service) CreateSeries(ctx context.Context, author, name, requestedSlug, description string) (*model.Series, error) {
	slug := slugify(requestedSlug)
	if requestedSlug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		return nil, blogerr.InvalidInput("slug_empty")
	}
	if existing, err := s.dao.GetSeriesBySlug(ctx, slug); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, blogerr.SlugTaken(slug)
	}
	m := &model.Series{ID: identifier.MustNew().String(), Slug: slug, Name: name, Description: description, AuthorID: author}
	if err := s.dao.InsertSeries(ctx, m); err != nil {
		return nil, err
	}
	return s.dao.GetSeriesByID(ctx, m.ID)
}

// ListSeries returns all series with their published-post counts.
func (s *Service) ListSeries(ctx context.Context) ([]*model.Series, error) {
	items, err := s.dao.ListSeries(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.dao.SeriesPostCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, x := range items {
		x.PostCount = counts[x.ID]
		if r, err := s.dao.RecentPostsBySeries(ctx, x.ID, 3); err == nil {
			x.Recent = r
		}
	}
	return items, nil
}

// SeriesWithPosts returns a series by slug plus its published posts in order.
func (s *Service) SeriesWithPosts(ctx context.Context, slug string) (*model.Series, []*model.Post, error) {
	se, err := s.dao.GetSeriesBySlug(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	if se == nil {
		return nil, nil, blogerr.NotFound(slug)
	}
	posts, err := s.dao.PostsBySeries(ctx, se.ID)
	if err != nil {
		return nil, nil, err
	}
	se.PostCount = len(posts)
	return se, posts, nil
}

// GetSeries returns a single series by id (for the detail page's series meta).
func (s *Service) GetSeries(ctx context.Context, id string) (*model.Series, error) {
	return s.dao.GetSeriesByID(ctx, id)
}

// ownedSeries fetches a series and asserts the caller owns it (or is admin).
func (s *Service) ownedSeries(ctx context.Context, author string, isAdmin bool, id string) (*model.Series, error) {
	se, err := s.dao.GetSeriesByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if se == nil || (se.AuthorID != author && !isAdmin) {
		return nil, blogerr.NotFound(id)
	}
	return se, nil
}

// UpdateSeries changes a series' public identity and description (owner or admin).
func (s *Service) UpdateSeries(ctx context.Context, author string, isAdmin bool, id string, name, requestedSlug, description *string) (*model.Series, error) {
	if _, err := s.ownedSeries(ctx, author, isAdmin, id); err != nil {
		return nil, err
	}
	fields := g.Map{}
	if name != nil && *name != "" {
		fields["name"] = *name
	}
	if requestedSlug != nil {
		slug := slugify(*requestedSlug)
		if slug == "" {
			return nil, blogerr.InvalidInput("slug_empty")
		}
		existing, err := s.dao.GetSeriesBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, blogerr.SlugTaken(slug)
		}
		fields["slug"] = slug
	}
	if description != nil {
		fields["description"] = *description
	}
	if len(fields) > 0 {
		if err := s.dao.UpdateSeries(ctx, id, fields); err != nil {
			return nil, err
		}
	}
	return s.dao.GetSeriesByID(ctx, id)
}

// DeleteSeries removes a series (owner or admin); member posts detach via FK.
func (s *Service) DeleteSeries(ctx context.Context, author string, isAdmin bool, id string) error {
	if _, err := s.ownedSeries(ctx, author, isAdmin, id); err != nil {
		return err
	}
	return s.dao.DeleteSeries(ctx, id)
}

// SetPostSeries assigns the author's post to a series at the given order (an
// empty seriesID clears it). The post must belong to the author (or admin); the
// target series, if any, must exist.
func (s *Service) SetPostSeries(ctx context.Context, author string, isAdmin bool, postID, seriesID string, order int) error {
	p, err := s.dao.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if p == nil || (p.AuthorID != author && !isAdmin) {
		return blogerr.NotFound(postID)
	}
	if seriesID != "" {
		se, err := s.dao.GetSeriesByID(ctx, seriesID)
		if err != nil {
			return err
		}
		if se == nil {
			return blogerr.InvalidInput("series_not_found")
		}
	}
	return s.dao.SetPostSeries(ctx, postID, seriesID, order)
}
