package monitor

import (
	"context"
	"errors"

	"encore.app/site"
	"encore.dev/pubsub"
	"encore.dev/storage/sqldb"
)

type TransitionEvent struct {
	Site *site.Site `json:"site"`
	Up   bool       `json:"up"`
}

var TransitionTopic = pubsub.NewTopic[*TransitionEvent]("uptime-transition", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

func getPreviousMeasurement(ctx context.Context, siteID int) (up bool, err error) {
	err = sqldb.QueryRow(ctx, `
	SELECT up FROM checks
	WHERE site_id = $1
	ORDER BY checked_at DESC
	LIMIT 1
	`, siteID).Scan(&up)
	if errors.Is(err, sqldb.ErrNoRows) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return up, nil
}

func publishOnTransition(ctx context.Context, site *site.Site, isUp bool) error {
	wasUp, err := getPreviousMeasurement(ctx, site.ID)
	if err != nil {
		return err
	}
	if isUp == wasUp {
		return nil
	}
	_, err = TransitionTopic.Publish(ctx, &TransitionEvent{
		Site: site, Up: isUp,
	})
	return err
}
