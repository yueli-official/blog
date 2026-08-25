package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

const initialHomeConfig = `
INSERT INTO home_config (
	key, eyebrow, title, subtitle,
	site_title, site_description, support_email,
	footer_tagline, footer_copyright
)
VALUES (
	'default', '月离博客', '把想法写成可以长期沉淀的内容',
	'文章、系列与讨论，在清晰的阅读体验中持续积累。',
	$1, $2, '',
	'记录值得反复阅读的想法。', ''
)
ON CONFLICT (key) DO NOTHING`

const localCatalog = `
INSERT INTO blog_classification_catalogs (id, catalog_key, revision)
VALUES ('019c52f0-1000-7000-8000-000000000001', 'blog', 1)
ON CONFLICT (catalog_key) DO NOTHING`

const localPolicy = `
INSERT INTO blog_classification_policy_profiles (
	catalog_id, policy_key, schema_version, policy_revision,
	category_policy, facet_policies, tag_policy, discovery_policy
)
SELECT id, 'blog.post.default', 1, 1,
	'{"minAssignments":0,"maxAssignments":0,"requirePrimary":false,"leafOnly":false,"maxDepth":0}'::jsonb,
	'[]'::jsonb,
	'{"unknown":"create","minAssignments":0,"maxAssignments":0}'::jsonb,
	'{"defaultSort":"name_asc"}'::jsonb
FROM blog_classification_catalogs
WHERE catalog_key = 'blog'
ON CONFLICT (catalog_id, policy_key) DO NOTHING`

const localCategories = `
INSERT INTO blog_categories (
	id, catalog_id, current_name, current_slug, description,
	status, editorial_position, first_activated_at
)
SELECT seed.id::uuid, catalog.id, seed.name, seed.slug, seed.description,
	'active', seed.position, TIMESTAMPTZ '2026-08-01 00:00:00+08'
FROM (VALUES
	('019c52f0-1000-7000-8000-000000000101', '写作方法', 'writing', '关于记录、整理与长期写作。', 10),
	('019c52f0-1000-7000-8000-000000000102', '产品与工程', 'product-engineering', '产品判断、工程实践与发布复盘。', 20),
	('019c52f0-1000-7000-8000-000000000103', '日常观察', 'daily-notes', '值得保存的生活片段与思考。', 30)
) AS seed(id, name, slug, description, position)
CROSS JOIN (SELECT id FROM blog_classification_catalogs WHERE catalog_key = 'blog') AS catalog
ON CONFLICT DO NOTHING`

const localTags = `
INSERT INTO blog_tags (
	id, catalog_id, current_name, current_slug, description,
	status, first_activated_at
)
SELECT seed.id::uuid, catalog.id, seed.name, seed.slug, seed.description,
	'active', TIMESTAMPTZ '2026-08-01 00:00:00+08'
FROM (VALUES
	('019c52f0-1000-7000-8000-000000000201', '写作', 'writing', '持续写作与内容整理。'),
	('019c52f0-1000-7000-8000-000000000202', '产品', 'product', '产品设计与发布判断。'),
	('019c52f0-1000-7000-8000-000000000203', '工程', 'engineering', '工程实践与质量保障。'),
	('019c52f0-1000-7000-8000-000000000204', '生活', 'life', '日常观察与个人记录。')
) AS seed(id, name, slug, description)
CROSS JOIN (SELECT id FROM blog_classification_catalogs WHERE catalog_key = 'blog') AS catalog
ON CONFLICT DO NOTHING`

const localTagLookups = `
INSERT INTO blog_tag_lookup_entries (
	catalog_id, lookup_key, target_tag_id, kind, display_value
)
SELECT catalog_id, current_name, id, 'canonical', current_name
FROM blog_tags
WHERE current_slug IN ('writing', 'product', 'engineering', 'life')
ON CONFLICT (catalog_id, lookup_key) DO NOTHING`

const localSeries = `
INSERT INTO series (id, slug, name, description, author_id, created_at, updated_at)
VALUES
	('019c52f0-1000-7000-8000-000000000301', 'sustainable-writing', '可持续写作', '从记录到发布，建立可以长期坚持的写作系统。', $1, TIMESTAMPTZ '2026-08-01 09:00:00+08', TIMESTAMPTZ '2026-08-01 09:00:00+08'),
	('019c52f0-1000-7000-8000-000000000302', 'shipping-notes', '发布手记', '产品上线前后的判断、检查与复盘。', $1, TIMESTAMPTZ '2026-08-02 09:00:00+08', TIMESTAMPTZ '2026-08-02 09:00:00+08')
ON CONFLICT DO NOTHING`

const localPosts = `
INSERT INTO posts (
	id, author_id, post_type, title, slug, content, excerpt,
	comment_status, status, locale, pinned, featured, series_id, series_order,
	published_at, created_at, updated_at
)
VALUES
	('019c52f0-1000-7000-8000-000000000401', $1, 'post',
	 '为什么要把想法写下来', 'why-write-things-down',
	 E'# 为什么要把想法写下来\n\n写作不是给记忆做备份，而是把模糊的判断变成可以检查、修正和继续积累的东西。\n\n## 从一条笔记开始\n\n先写下此刻真正想解决的问题，再补充证据与下一步。',
	 '写作让模糊的判断变得可以检查，也让零散经验成为能继续积累的内容。',
	 1, 'published', 'zh-CN', true, true,
	 (SELECT id FROM series WHERE slug = 'sustainable-writing'), 1,
	 TIMESTAMPTZ '2026-08-16 09:30:00+08', TIMESTAMPTZ '2026-08-16 09:30:00+08', TIMESTAMPTZ '2026-08-16 09:30:00+08'),
	('019c52f0-1000-7000-8000-000000000402', $1, 'post',
	 '从零搭起可持续的写作系统', 'build-a-sustainable-writing-system',
	 E'# 从零搭起可持续的写作系统\n\n一个可靠的写作系统只需要三个阶段：捕捉、整理和发布。工具应该帮助内容流动，而不是制造更多待办。\n\n## 保持节奏\n\n固定一个足够小的发布频率，让反馈成为下一篇文章的输入。',
	 '用捕捉、整理和发布三个阶段，建立不依赖灵感的长期写作节奏。',
	 1, 'published', 'zh-CN', false, true,
	 (SELECT id FROM series WHERE slug = 'sustainable-writing'), 2,
	 TIMESTAMPTZ '2026-08-13 14:00:00+08', TIMESTAMPTZ '2026-08-13 14:00:00+08', TIMESTAMPTZ '2026-08-13 14:00:00+08'),
	('019c52f0-1000-7000-8000-000000000403', $1, 'post',
	 '一次产品发布前的检查清单', 'a-preflight-checklist-for-shipping',
	 E'# 一次产品发布前的检查清单\n\n发布前最后一轮检查应该围绕真实用户旅程，而不是只确认构建命令成功。\n\n- 核心入口可以到达\n- 登录与权限边界清楚\n- 写入可以保存、刷新与恢复\n- 错误状态能解释下一步',
	 '把发布前检查收敛到真实入口、权限、写入恢复和错误状态四件事。',
	 1, 'published', 'zh-CN', false, true,
	 (SELECT id FROM series WHERE slug = 'shipping-notes'), 1,
	 TIMESTAMPTZ '2026-08-10 10:20:00+08', TIMESTAMPTZ '2026-08-10 10:20:00+08', TIMESTAMPTZ '2026-08-10 10:20:00+08'),
	('019c52f0-1000-7000-8000-000000000404', $1, 'post',
	 '在代码与生活之间保留余白', 'leave-room-between-code-and-life',
	 E'# 在代码与生活之间保留余白\n\n真正有价值的观察通常发生在离开屏幕之后。散步、阅读和没有目标的停顿，会让已经拥挤的问题重新显出结构。',
	 '偶尔离开屏幕，让已经拥挤的问题重新显出结构。',
	 1, 'published', 'zh-CN', false, false, NULL, 0,
	 TIMESTAMPTZ '2026-08-06 18:10:00+08', TIMESTAMPTZ '2026-08-06 18:10:00+08', TIMESTAMPTZ '2026-08-06 18:10:00+08'),
	('019c52f0-1000-7000-8000-000000000405', $1, 'post',
	 '把复杂问题拆成可以验证的步骤', 'turn-complex-problems-into-verifiable-steps',
	 E'# 把复杂问题拆成可以验证的步骤\n\n面对复杂问题时，先找到一个可以在几分钟内给出红绿结论的反馈点。可靠的小循环会让后续判断越来越便宜。',
	 '先建立能快速给出红绿结论的反馈点，再逐步缩小复杂问题。',
	 1, 'published', 'zh-CN', false, false,
	 (SELECT id FROM series WHERE slug = 'shipping-notes'), 2,
	 TIMESTAMPTZ '2026-08-04 11:00:00+08', TIMESTAMPTZ '2026-08-04 11:00:00+08', TIMESTAMPTZ '2026-08-04 11:00:00+08'),
	('019c52f0-1000-7000-8000-000000000406', $1, 'post',
	 '给长期项目留一份清晰的工作台', 'keep-a-clear-desk-for-long-running-work',
	 E'# 给长期项目留一份清晰的工作台\n\n长期工作最怕每次回来都重新发现问题。把当前事实、下一步和验证证据放在同一个小工作台里，恢复就会变得可靠。',
	 '用当前事实、下一步和验证证据，让长期工作随时可以恢复。',
	 1, 'published', 'zh-CN', false, false,
	 (SELECT id FROM series WHERE slug = 'sustainable-writing'), 3,
	 TIMESTAMPTZ '2026-08-03 16:30:00+08', TIMESTAMPTZ '2026-08-03 16:30:00+08', TIMESTAMPTZ '2026-08-03 16:30:00+08'),
	('019c52f0-1000-7000-8000-000000000407', $1, 'post',
	 '一次失败发布带来的三个提醒', 'three-lessons-from-a-failed-release',
	 E'# 一次失败发布带来的三个提醒\n\n构建成功不等于用户旅程可用。发布前要验证真实数据、身份跳转和失败后的恢复路径，并把结论留给下一次发布。',
	 '构建、身份和恢复路径，缺一项都不能算完成发布验收。',
	 1, 'published', 'zh-CN', false, false,
	 (SELECT id FROM series WHERE slug = 'shipping-notes'), 3,
	 TIMESTAMPTZ '2026-08-02 13:40:00+08', TIMESTAMPTZ '2026-08-02 13:40:00+08', TIMESTAMPTZ '2026-08-02 13:40:00+08'),
	('019c52f0-1000-7000-8000-000000000408', $1, 'post',
	 '周末散步时重新理解了节奏', 'rethinking-rhythm-on-a-weekend-walk',
	 E'# 周末散步时重新理解了节奏\n\n节奏不是把每一分钟排满，而是知道什么时候推进、什么时候停下来观察。留白让下一次行动更准确。',
	 '节奏来自推进与停顿之间清楚的切换，而不是排满时间。',
	 1, 'published', 'zh-CN', false, false, NULL, 0,
	 TIMESTAMPTZ '2026-08-01 17:20:00+08', TIMESTAMPTZ '2026-08-01 17:20:00+08', TIMESTAMPTZ '2026-08-01 17:20:00+08')
ON CONFLICT DO NOTHING`

const localPostStats = `
INSERT INTO post_stats (post_id, view_count, like_count, comment_count, share_count)
SELECT post.id, seed.views, seed.likes, 0, seed.shares
FROM (VALUES
	('why-write-things-down', 286::bigint, 24::bigint, 8::bigint),
	('build-a-sustainable-writing-system', 198::bigint, 17::bigint, 5::bigint),
	('a-preflight-checklist-for-shipping', 164::bigint, 12::bigint, 4::bigint),
	('leave-room-between-code-and-life', 93::bigint, 9::bigint, 2::bigint),
	('turn-complex-problems-into-verifiable-steps', 82::bigint, 8::bigint, 2::bigint),
	('keep-a-clear-desk-for-long-running-work', 76::bigint, 7::bigint, 2::bigint),
	('three-lessons-from-a-failed-release', 68::bigint, 6::bigint, 1::bigint),
	('rethinking-rhythm-on-a-weekend-walk', 57::bigint, 5::bigint, 1::bigint)
) AS seed(slug, views, likes, shares)
JOIN posts AS post ON post.slug = seed.slug
ON CONFLICT (post_id) DO NOTHING`

const localPostSEO = `
INSERT INTO post_seo (post_id, meta_title, meta_desc)
SELECT id, title, excerpt
FROM posts
WHERE slug IN (
	'why-write-things-down',
	'build-a-sustainable-writing-system',
	'a-preflight-checklist-for-shipping',
	'leave-room-between-code-and-life',
	'turn-complex-problems-into-verifiable-steps',
	'keep-a-clear-desk-for-long-running-work',
	'three-lessons-from-a-failed-release',
	'rethinking-rhythm-on-a-weekend-walk'
)
ON CONFLICT (post_id) DO NOTHING`

const localCategoryAssignments = `
INSERT INTO blog_post_category_assignments (post_id, category_id)
SELECT post.id, category.id
FROM (VALUES
	('why-write-things-down', 'writing'),
	('build-a-sustainable-writing-system', 'writing'),
	('a-preflight-checklist-for-shipping', 'product-engineering'),
	('leave-room-between-code-and-life', 'daily-notes'),
	('turn-complex-problems-into-verifiable-steps', 'product-engineering'),
	('keep-a-clear-desk-for-long-running-work', 'writing'),
	('three-lessons-from-a-failed-release', 'product-engineering'),
	('rethinking-rhythm-on-a-weekend-walk', 'daily-notes')
) AS seed(post_slug, category_slug)
JOIN posts AS post ON post.slug = seed.post_slug
JOIN blog_categories AS category ON category.current_slug = seed.category_slug
ON CONFLICT DO NOTHING`

const localTagAssignments = `
INSERT INTO blog_post_tag_assignments (post_id, tag_id)
SELECT post.id, tag.id
FROM (VALUES
	('why-write-things-down', 'writing'),
	('build-a-sustainable-writing-system', 'writing'),
	('build-a-sustainable-writing-system', 'engineering'),
	('a-preflight-checklist-for-shipping', 'product'),
	('a-preflight-checklist-for-shipping', 'engineering'),
	('leave-room-between-code-and-life', 'life'),
	('turn-complex-problems-into-verifiable-steps', 'engineering'),
	('keep-a-clear-desk-for-long-running-work', 'writing'),
	('three-lessons-from-a-failed-release', 'product'),
	('three-lessons-from-a-failed-release', 'engineering'),
	('rethinking-rhythm-on-a-weekend-walk', 'life')
) AS seed(post_slug, tag_slug)
JOIN posts AS post ON post.slug = seed.post_slug
JOIN blog_tags AS tag ON tag.current_slug = seed.tag_slug
ON CONFLICT DO NOTHING`

const localTrafficSources = `
INSERT INTO blog_traffic_source_daily (day, source, views)
SELECT CURRENT_DATE - seed.days_ago, seed.source, seed.views
FROM (VALUES
	(0, 'direct', 5::bigint),
	(1, 'google.com', 6::bigint),
	(2, 'direct', 4::bigint),
	(3, 'zhihu.com', 5::bigint),
	(4, 'google.com', 4::bigint),
	(5, 'x.com', 3::bigint),
	(6, 'direct', 6::bigint),
	(7, 'google.com', 5::bigint),
	(8, 'internal', 4::bigint),
	(9, 'zhihu.com', 3::bigint),
	(10, 'direct', 5::bigint),
	(11, 'google.com', 4::bigint),
	(12, 'bing.com', 2::bigint),
	(13, 'direct', 3::bigint)
) AS seed(days_ago, source, views)
ON CONFLICT (day, source) DO NOTHING`

const removeLocalSubscriberSamples = `
DELETE FROM subscribers
WHERE id IN (
	'019c52f0-1000-7000-8000-000000000501',
	'019c52f0-1000-7000-8000-000000000502',
	'019c52f0-1000-7000-8000-000000000503',
	'019c52f0-1000-7000-8000-000000000504',
	'019c52f0-1000-7000-8000-000000000505',
	'019c52f0-1000-7000-8000-000000000506'
)`

func installLocalAcceptanceContent(ctx context.Context, tx *sql.Tx, authorSub string) error {
	steps := []struct {
		name  string
		query string
		args  []any
	}{
		{name: "classification catalog", query: localCatalog},
		{name: "classification policy", query: localPolicy},
		{name: "categories", query: localCategories},
		{name: "tags", query: localTags},
		{name: "tag lookups", query: localTagLookups},
		{name: "series", query: localSeries, args: []any{authorSub}},
		{name: "posts", query: localPosts, args: []any{authorSub}},
		{name: "post stats", query: localPostStats},
		{name: "post SEO", query: localPostSEO},
		{name: "category assignments", query: localCategoryAssignments},
		{name: "tag assignments", query: localTagAssignments},
		{name: "subscriber sample cleanup", query: removeLocalSubscriberSamples},
		{name: "traffic sources", query: localTrafficSources},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.args...); err != nil {
			return fmt.Errorf("install local acceptance %s: %w", step.name, err)
		}
	}
	return nil
}

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("BLOG_DATABASE_URL"))
	if databaseURL == "" {
		fail("BLOG_DATABASE_URL is required")
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		fail("open database: %v", err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := database.PingContext(ctx); err != nil {
		fail("connect database: %v", err)
	}
	brand := strings.TrimSpace(os.Getenv("BLOG_SITE_BRAND"))
	if brand == "" {
		brand = "月离博客"
	}
	description := strings.TrimSpace(os.Getenv("BLOG_SITE_DESCRIPTION"))
	if description == "" {
		description = "想法、笔记与记录"
	}
	if _, err := database.ExecContext(ctx, initialHomeConfig, brand, description); err != nil {
		fail("install initial Blog configuration: %v", err)
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("BLOG_DEV_SEED")), "true") {
		authorSub := strings.TrimSpace(os.Getenv("BLOG_DEV_SEED_AUTHOR_SUB"))
		if authorSub == "" {
			fail("BLOG_DEV_SEED_AUTHOR_SUB is required when BLOG_DEV_SEED=true")
		}
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			fail("begin local acceptance seed: %v", err)
		}
		if err := installLocalAcceptanceContent(ctx, tx, authorSub); err != nil {
			_ = tx.Rollback()
			fail("%v", err)
		}
		if err := tx.Commit(); err != nil {
			fail("commit local acceptance seed: %v", err)
		}
	}
	fmt.Println("Blog initial records are ready")
}

func fail(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, "bootstrap: "+format+"\n", arguments...)
	os.Exit(1)
}
