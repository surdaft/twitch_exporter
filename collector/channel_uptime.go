package collector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/damoun/twitch_exporter/internal/eventsub"
	"github.com/nicklaw5/helix/v2"
	"github.com/prometheus/client_golang/prometheus"
)

type channelUpTimeCollector struct {
	logger       *slog.Logger
	client       *helix.Client
	channelNames ChannelNames

	channelUpTime typedDesc
}

var (
	// firstPush
	firstPush = map[string]bool{}
	// firstPushMutex is used to lock the firstPush and avoid concurrent map writes
	firstPushMutex = sync.Mutex{}
	// lastScrape is the last time we scraped the uptime
	lastScrape = time.Now()
)

func init() {
	registerCollector("channel_uptime_total", defaultEnabled, NewChannelUpTimeCollector)
}

func NewChannelUpTimeCollector(logger *slog.Logger, client *helix.Client, eventsubClient *eventsub.Client, channelNames ChannelNames) (Collector, error) {
	c := channelUpTimeCollector{
		logger:       logger,
		client:       client,
		channelNames: channelNames,

		channelUpTime: typedDesc{prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "channel_uptime_total"),
			"Channel uptime total in seconds.",
			[]string{"username"}, nil,
		), prometheus.UntypedValue},
	}

	return c, nil
}

func (c channelUpTimeCollector) Update(ch chan<- prometheus.Metric) error {
	if len(c.channelNames) == 0 {
		return ErrNoData
	}

	streamsResp, err := c.client.GetStreams(&helix.StreamsParams{
		UserLogins: c.channelNames,
		First:      len(c.channelNames),
	})

	if err != nil {
		c.logger.Error("could not get streams", "err", err)
		return err
	}

	// loop through the live streams and push the uptime, if it is the first time we are pushing
	// the uptime for this channel since they went online
	// note: potential issue if the collector is restarted, since we may increment the uptime
	// multiple times
	wg := sync.WaitGroup{}
	for _, s := range streamsResp.Data.Streams {
		wg.Add(1)

		go c.pushUptime(ch, s.UserName, time.Since(s.StartedAt), func() {
			wg.Done()
		})
	}

	wg.Wait()

	lastScrape = time.Now()
	return nil
}

func (c channelUpTimeCollector) pushUptime(ch chan<- prometheus.Metric, username string, uptime time.Duration, done func()) {
	ch <- prometheus.MustNewConstMetric(
		c.channelUpTime.desc,
		prometheus.CounterValue,
		uptime.Seconds(),
		username,
	)

	done()
}
