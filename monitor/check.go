package monitor

import (
	"context"

	"encore.app/site"
	"encore.dev/cron"
	"encore.dev/storage/sqldb"
	"golang.org/x/sync/errgroup"
)

// Check checks a single site.
//
//encore:api public method=POST path=/check/:siteID
func Check(ctx context.Context, siteID int) error {
	site, err := site.Get(ctx, siteID)
	if err != nil {
		return err
	}
	return check(ctx, site)
}

// CheckAll checks all sites.
//
//encore:api public method=POST path=/checkall
func CheckAll(ctx context.Context) error {
	resp, err := site.List(ctx)
	if err != nil {
		return err
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for _, site := range resp.Sites {
		site := site
		g.Go(func() error {
			return check(ctx, site)
		})
	}
	return g.Wait()
}

func check(ctx context.Context, site *site.Site) error {
	result, err := Ping(ctx, site.URL)
	if err != nil {
		return err
	}

	_, err = sqldb.Exec(ctx, `INSERT INTO checks (site_id,up,checked_at)
	VALUES ($1,$2,NOW())`, site.ID, result.Up)
	return err
}

var _ = cron.NewJob("check-all", cron.JobConfig{
	Title:    "Check all sites",
	Endpoint: CheckAll,
	Every:    1 * cron.Hour,
})
