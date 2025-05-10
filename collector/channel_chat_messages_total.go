package collector

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/damoun/twitch_exporter/internal/eventsub"
	"github.com/nicklaw5/helix/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	chatMessages      = MessageCounter{}
	chatMessagesMutex = sync.Mutex{}
)

type MessageCounter map[string]int

func (m MessageCounter) Add(username string) {
	chatMessagesMutex.Lock()
	chatMessages[username]++
	chatMessagesMutex.Unlock()
}

func (m MessageCounter) Reset(username string) {
	chatMessagesMutex.Lock()
	chatMessages[username] = 0
	chatMessagesMutex.Unlock()
}

type ChannelChatMessagesCollector struct {
	logger       *slog.Logger
	client       *helix.Client
	channelNames ChannelNames

	channelChatMessages typedDesc
}

func init() {
	// disabled by default since you need to use webhooks to listen for events using an app access token
	// which requires it to be exposed to the internet
	registerCollector("channel_chat_messages_total", defaultDisabled, NewChannelChatMessagesCollector)
}

func NewChannelChatMessagesCollector(logger *slog.Logger, client *helix.Client, channelNames ChannelNames) (Collector, error) {
	// this means that eventsub.enabled must be true, otherwise the default client will not be set
	if eventsub.DefaultClient == nil {
		return nil, eventsub.ErrEventsubDefaultClientNotSet
	}

	err := eventsub.On("channel.chat.message", func(eventRaw json.RawMessage) {
		var event eventsub.ChannelChatMessageEvent

		if err := json.Unmarshal(eventRaw, &event); err != nil {
			logger.Error("failed to unmarshal channel chat message event", "error", err)
			return
		}

		chatMessages.Add(event.Event.ChatterUserName)
	})

	// in theory the only error this could be is ErrEventsubDefaultClientNotSet which is already handled
	// but it returns an error in case that expands
	if err != nil {
		return nil, err
	}

	c := ChannelChatMessagesCollector{
		logger:       logger,
		client:       client,
		channelNames: channelNames,

		channelChatMessages: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "channel_chat_messages_total"),
			"The number of chat messages sent in a channel.",
			[]string{"username"}, nil,
		), prometheus.GaugeValue},
	}

	return c, nil
}

func (c ChannelChatMessagesCollector) Update(ch chan<- prometheus.Metric) error {
	if len(c.channelNames) == 0 {
		return ErrNoData
	}

	// loop all the channels and push the counts we have collected since last scrape
	for username, count := range chatMessages {
		ch <- prometheus.MustNewConstMetric(c.channelChatMessages.desc, prometheus.GaugeValue, float64(count), username)

		// reset the counter
		chatMessages.Reset(username)
	}

	return nil
}
