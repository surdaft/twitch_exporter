package collector

import (
	"log/slog"
	"sync"
	"time"

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

func NewChannelUpTimeCollector(logger *slog.Logger, client *helix.Client, channelNames ChannelNames) (Collector, error) {
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
	for _, s := range streamsResp.Data.Streams {
		if _, ok := firstPush[s.UserName]; !ok {
			firstPushMutex.Lock()

			firstPush[s.UserName] = true
			ch <- prometheus.MustNewConstMetric(
				c.channelUpTime.desc,
				prometheus.CounterValue,
				time.Since(s.StartedAt).Seconds(),
				s.UserName,
			)

			firstPushMutex.Unlock()
		} else {
			// since we have already done the first push, we just increment the time
			// since the last scrape
			ch <- prometheus.MustNewConstMetric(
				c.channelUpTime.desc,
				prometheus.CounterValue,
				time.Since(lastScrape).Seconds(),
				s.UserName,
			)
		}
	}

	lastScrape = time.Now()
	return nil
}
