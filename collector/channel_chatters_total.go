package collector

import (
	"errors"
	"log/slog"

	"github.com/damoun/twitch_exporter/internal/eventsub"
	"github.com/nicklaw5/helix/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type channelChattersTotalCollector struct {
	logger       *slog.Logger
	client       *helix.Client
	channelNames ChannelNames
	channelIDmap map[string]string

	channelChatters typedDesc
}

func init() {
	// note: this will only work if the user is a moderator of the channel, so we need to
	// figure out who owns the access token and then request with that user as the moderator
	// for that reason we disable this collector by default
	registerCollector("channel_chatters_total", defaultDisabled, NewChannelChattersTotalCollector)
}

func NewChannelChattersTotalCollector(logger *slog.Logger, client *helix.Client, _ *eventsub.Client, channelNames ChannelNames) (Collector, error) {
	c := channelChattersTotalCollector{
		logger:       logger,
		client:       client,
		channelNames: channelNames,
		channelIDmap: make(map[string]string),

		channelChatters: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "channel_chatters_total"),
			"The number of chatters in a channel.",
			[]string{"username"}, nil,
		), prometheus.GaugeValue},
	}

	users, err := client.GetUsers(&helix.UsersParams{
		Logins: channelNames,
	})
	if err != nil {
		return nil, err
	}

	// we will need this later to get the chatters, it does not allow for usernames only IDs
	for _, user := range users.Data.Users {
		c.channelIDmap[user.Login] = user.ID
	}

	return c, nil
}

func (c channelChattersTotalCollector) Update(ch chan<- prometheus.Metric) error {
	if len(c.channelNames) == 0 {
		return ErrNoData
	}

	for _, channelName := range c.channelNames {
		// todo: technically this is paginated, however there is a "total" field returned that
		// allows us to bypass pagination. would we need to handle pagination in the future?
		chattersResp, err := c.client.GetChannelChatChatters(&helix.GetChatChattersParams{
			BroadcasterID: c.channelIDmap[channelName],
			ModeratorID:   c.channelIDmap[channelName],
		})

		if err != nil {
			c.logger.Error("Failed to collect chatter stats from Twitch helix API", "err", err)
			return err
		}

		if chattersResp.StatusCode != 200 {
			c.logger.Error("Failed to collect chatter stats from Twitch helix API", "err", chattersResp.ErrorMessage)
			return errors.New(chattersResp.ErrorMessage)
		}

		ch <- c.channelChatters.mustNewConstMetric(float64(chattersResp.Data.Total), channelName)
	}

	return nil
}
