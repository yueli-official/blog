package blogdiscovery

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yueli-official/foundation/go/discovery"
	"github.com/yuin/goldmark"

	"github.com/yueli-official/blog/api/internal/dao"
	"github.com/yueli-official/blog/api/internal/model"
)

type Config struct {
	Origin      string
	Name        string
	Description string
	Locale      string
	TTL         time.Duration
	Clock       func() time.Time
}

func New(store *dao.PG, config Config) (*discovery.Module, *discovery.Cache, error) {
	module, err := discovery.Compile(discovery.Definition{
		ContractVersion: discovery.ContractVersion,
		Site: discovery.SiteProfile{
			Origin: config.Origin, Name: config.Name,
			Description: config.Description, DefaultLocale: config.Locale,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	cache, err := discovery.NewCache(module, discovery.CacheOptions{
		TTL: config.TTL, Clock: config.Clock,
		Build: func(ctx context.Context) (discovery.PublicationPlan, discovery.Sources, error) {
			return buildPublication(ctx, module, store, config)
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return module, cache, nil
}

func ProjectPost(
	module *discovery.Module,
	post *model.Post,
	seo *model.SEO,
	authorName string,
) (discovery.PageProjection, error) {
	if module == nil || post == nil || post.Status != model.StatusPublished {
		return discovery.PageProjection{}, fmt.Errorf("published post and Discovery module are required")
	}
	title, description, imageURL, pagePath, robots := post.Title, post.Excerpt, post.CoverURL, "/posts/"+post.Slug, ""
	if seo != nil {
		if seo.MetaTitle != "" {
			title = seo.MetaTitle
		}
		if seo.MetaDesc != "" {
			description = seo.MetaDesc
		}
		if seo.OgImage != "" {
			imageURL = seo.OgImage
		}
		if seo.CanonicalURL != "" {
			pagePath = seo.CanonicalURL
		}
		robots = seo.Robots
	}
	if description == "" {
		description = excerpt(post.Content, 300)
	}
	if authorName == "" {
		authorName = post.AuthorID
	}
	publishedAt := time.Time{}
	if post.PublishedAt != nil {
		publishedAt = post.PublishedAt.Time.UTC()
	}
	var modifiedAt *time.Time
	if post.UpdatedAt != nil {
		value := post.UpdatedAt.Time.UTC()
		modifiedAt = &value
	}
	var image *discovery.Image
	if imageURL != "" {
		image = &discovery.Image{URL: imageURL, Alt: title}
	}
	visibility, follow := robotsPolicy(robots)
	projection, _, err := module.Project(discovery.PageDescriptor{
		Key: "post:" + post.ID, Path: pagePath,
		Locale: post.Locale, Visibility: visibility, Follow: follow,
		Subject: discovery.ArticleSubject(discovery.Article{
			Title: title, Description: description, Image: image,
			PublishedAt: publishedAt, ModifiedAt: modifiedAt,
			Authors: []discovery.Person{{
				Name: authorName, URL: "/author/" + post.AuthorID,
			}},
		}),
	})
	return projection, err
}

type source struct {
	module       *discovery.Module
	store        *dao.PG
	origin       string
	siteName     string
	siteDesc     string
	locale       string
	includeHome  bool
	taxonomyKind string
	taxonomySlug string
	feed         bool
}

func (value *source) Next(ctx context.Context, cursor discovery.Cursor, limit int) (discovery.Batch, error) {
	after := string(cursor)
	rowsLimit := limit + 1
	records := make([]discovery.Record, 0, limit)
	if value.includeHome && after == "" {
		projection, _, err := value.module.Project(discovery.PageDescriptor{
			Key: "site:home", Path: "/", Locale: value.locale,
			Subject: discovery.WebPageSubject(discovery.WebPage{
				Title: value.siteName, Description: value.siteDesc,
			}),
		})
		if err != nil {
			return discovery.Batch{}, err
		}
		records = append(records, discovery.Record{
			SortKey: projection.CanonicalURL,
			Page: discovery.PageDescriptor{
				Key: "site:home", Path: "/", Locale: value.locale,
				Subject: discovery.WebPageSubject(discovery.WebPage{
					Title: value.siteName, Description: value.siteDesc,
				}),
			},
		})
		after = projection.CanonicalURL
		rowsLimit = limit
	}
	var rows []dao.DiscoveryRow
	var err error
	if value.feed {
		rows, err = value.store.ListDiscoveryFeedPosts(ctx, value.origin, after, value.taxonomyKind, value.taxonomySlug, rowsLimit)
	} else {
		rows, err = value.store.ListDiscoveryPages(ctx, value.origin, after, rowsLimit)
	}
	if err != nil {
		return discovery.Batch{}, err
	}
	hasMore := len(rows) >= rowsLimit
	if hasMore {
		rows = rows[:rowsLimit-1]
	}
	for _, row := range rows {
		record, err := value.record(row)
		if err != nil {
			return discovery.Batch{}, err
		}
		records = append(records, record)
		if len(records) == limit {
			hasMore = true
			break
		}
	}
	next := discovery.Cursor("")
	if len(records) > 0 {
		last := records[len(records)-1]
		next = discovery.Cursor(last.SortKey)
	}
	return discovery.Batch{Records: records, NextCursor: next, Done: !hasMore}, nil
}

func (value *source) record(row dao.DiscoveryRow) (discovery.Record, error) {
	pagePath := row.Path
	if row.CanonicalURL != "" {
		pagePath = row.CanonicalURL
	}
	visibility, follow := robotsPolicy(row.Robots)
	var modifiedAt *time.Time
	if row.UpdatedAt != nil {
		updated := row.UpdatedAt.Time.UTC()
		modifiedAt = &updated
	}
	var image *discovery.Image
	if row.ImageURL != "" {
		image = &discovery.Image{URL: row.ImageURL, Alt: row.Title}
	}
	var subject discovery.Subject
	if row.Kind == "article" {
		published := time.Time{}
		if row.PublishedAt != nil {
			published = row.PublishedAt.Time.UTC()
		}
		subject = discovery.ArticleSubject(discovery.Article{
			Title: row.Title, Description: row.Description, Image: image,
			PublishedAt: published, ModifiedAt: modifiedAt,
			Authors: []discovery.Person{{Name: row.AuthorID, URL: "/author/" + row.AuthorID}},
		})
	} else {
		subject = discovery.CollectionSubject(discovery.Collection{
			Title: row.Title, Description: row.Description, Image: image,
		})
	}
	page := discovery.PageDescriptor{
		Key: row.Key, Path: pagePath, Locale: row.Locale,
		Visibility: visibility, Follow: follow, Subject: subject,
	}
	projection, _, err := value.module.Project(page)
	if err != nil {
		return discovery.Record{}, err
	}
	record := discovery.Record{SortKey: projection.CanonicalURL, Page: page}
	if value.feed {
		published := time.Time{}
		if row.PublishedAt != nil {
			published = row.PublishedAt.Time.UTC()
		}
		record.Feed = &discovery.FeedEntryFacts{
			ID: "urn:yueli:blog:" + row.Key, Title: row.Title,
			Summary: row.Description, ContentHTML: markdown(row.Content),
			PublishedAt: published, ModifiedAt: modifiedAt,
			Authors: []discovery.Person{{Name: row.AuthorID, URL: "/author/" + row.AuthorID}},
		}
	}
	return record, nil
}

func buildPublication(
	ctx context.Context,
	module *discovery.Module,
	store *dao.PG,
	config Config,
) (discovery.PublicationPlan, discovery.Sources, error) {
	updatedAt, err := store.DiscoveryUpdatedAt(ctx)
	if err != nil {
		return discovery.PublicationPlan{}, nil, err
	}
	taxonomies, err := store.ListTaxonomies(ctx, "")
	if err != nil {
		return discovery.PublicationPlan{}, nil, err
	}
	sources := discovery.Sources{
		"pages": &source{
			module: module, store: store, origin: config.Origin,
			siteName: config.Name, siteDesc: config.Description,
			locale: config.Locale, includeHome: true,
		},
		"feed:all": &source{
			module: module, store: store, origin: config.Origin,
			locale: config.Locale, feed: true,
		},
	}
	plan := discovery.PublicationPlan{
		Sitemap: &discovery.SitemapPlan{Source: "pages"},
		Feeds: []discovery.FeedPlan{{
			ID: "urn:yueli:blog:feed", Source: "feed:all", Format: discovery.FeedRSS,
			Route: "rss.xml", Title: config.Name, Description: config.Description,
			Language: config.Locale, UpdatedAt: updatedAt, MaxEntries: 100,
		}},
		Robots: &discovery.RobotsPlan{},
	}
	for _, taxonomy := range taxonomies {
		sourceID := discovery.SourceID("feed:" + taxonomy.Taxonomy + ":" + taxonomy.Slug)
		sources[sourceID] = &source{
			module: module, store: store, origin: config.Origin,
			locale: config.Locale, feed: true,
			taxonomyKind: taxonomy.Taxonomy, taxonomySlug: taxonomy.Slug,
		}
		routePrefix := "categories"
		if taxonomy.Taxonomy == "tag" {
			routePrefix = "tags"
		}
		plan.Feeds = append(plan.Feeds, discovery.FeedPlan{
			ID:     "urn:yueli:blog:feed:" + taxonomy.Taxonomy + ":" + taxonomy.ID,
			Source: sourceID, Format: discovery.FeedRSS,
			Route:       routePrefix + "/" + taxonomy.Slug + ".xml",
			Title:       config.Name + " · " + taxonomy.Name,
			Description: taxonomy.Description, Language: config.Locale,
			UpdatedAt: updatedAt, MaxEntries: 100,
		})
	}
	return plan, sources, nil
}

func robotsPolicy(value string) (discovery.Visibility, discovery.FollowPolicy) {
	lower := strings.ToLower(value)
	visibility := discovery.Discoverable
	if strings.Contains(lower, "noindex") {
		visibility = discovery.Unlisted
	}
	follow := discovery.Follow
	if strings.Contains(lower, "nofollow") {
		follow = discovery.NoFollow
	}
	return visibility, follow
}

func excerpt(value string, max int) string {
	runes := []rune(strings.Join(strings.Fields(value), " "))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "…"
}

func markdown(value string) string {
	var output bytes.Buffer
	if err := goldmark.Convert([]byte(value), &output); err != nil {
		return ""
	}
	return output.String()
}
