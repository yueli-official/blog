package dao

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
)

type DashboardTrafficSource struct {
	Source string `orm:"source"`
	Views  int64  `orm:"views"`
}

func (p *PG) RecordTrafficSource(ctx context.Context, eventID, day, source string) error {
	return p.db.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		result, err := tx.Ctx(ctx).Exec(`
INSERT INTO blog_traffic_source_receipts (event_id, day, source)
VALUES (?, ?::date, ?)
ON CONFLICT (event_id) DO NOTHING`, eventID, day, source)
		if err != nil {
			return err
		}
		inserted, err := result.RowsAffected()
		if err != nil || inserted == 0 {
			return err
		}
		_, err = tx.Ctx(ctx).Exec(`
INSERT INTO blog_traffic_source_daily (day, source, views)
VALUES (?::date, ?, 1)
ON CONFLICT (day, source) DO UPDATE
SET views = blog_traffic_source_daily.views + 1`, day, source)
		return err
	})
}

func (p *PG) DashboardTrafficSources(
	ctx context.Context,
	from string,
	to string,
	limit int,
) ([]DashboardTrafficSource, error) {
	var sources []DashboardTrafficSource
	err := p.db.Ctx(ctx).Raw(`
SELECT source, SUM(views)::bigint AS views
FROM blog_traffic_source_daily
WHERE day >= ?::date AND day < ?::date AND source <> 'internal'
GROUP BY source
ORDER BY views DESC, source ASC
LIMIT ?`, from, to, limit).Scan(&sources)
	return sources, err
}

func (p *PG) PruneTrafficSourceReceipts(ctx context.Context, before string) error {
	_, err := p.db.Exec(ctx,
		"DELETE FROM blog_traffic_source_receipts WHERE day < ?::date",
		before,
	)
	return err
}
