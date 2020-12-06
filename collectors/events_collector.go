package collectors

import (
	"fmt"
	"net/url"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type EventsCollector struct {
	namespace                             string
	environment                           string
	deployment                            string
	cfClient                              *cfclient.Client
	eventsQuery                           string
	eventsInfoMetric                      *prometheus.GaugeVec
	eventsScrapesTotalMetric              prometheus.Counter
	eventsScrapeErrorsTotalMetric         prometheus.Counter
	lastEventsScrapeErrorMetric           prometheus.Gauge
	lastEventsScrapeTimestampMetric       prometheus.Gauge
	lastEventsScrapeDurationSecondsMetric prometheus.Gauge

	lastCheckFilter *string
	timeLocation    *time.Location
}

func NewEventsCollector(
	namespace string,
	environment string,
	deployment string,
	cfClient *cfclient.Client,
	eventsQuery string,
) *EventsCollector {
	eventsInfoMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "events",
			Name:        "info",
			Help:        "Labeled Cloud Foundry Events information with a constant '1' value.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
		[]string{"event_id","create_time","type", "actor", "actor_type", "actor_name", "actor_username", "actee", "actee_type", "actee_name", "space_id", "organization_id"},
	)

	eventsScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "events_scrapes",
			Name:        "total",
			Help:        "Total number of scrapes for Cloud Foundry Events.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	eventsScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "events_scrape_errors",
			Name:        "total",
			Help:        "Total number of scrape errors of Cloud Foundry Events.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastEventsScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_events_scrape_error",
			Help:        "Whether the last scrape of Event metrics from Cloud Foundry resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastEventsScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_events_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Event metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastEventsScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_events_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Events metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	timeLocation, _ := time.LoadLocation("UTC")
	newTime := time.Now().In(timeLocation).Format("2006-01-02T15:04:05Z")

	return &EventsCollector{
		namespace:                             namespace,
		environment:                           environment,
		eventsQuery:                           eventsQuery,
		deployment:                            deployment,
		cfClient:                              cfClient,
		eventsInfoMetric:                      eventsInfoMetric,
		eventsScrapesTotalMetric:              eventsScrapesTotalMetric,
		eventsScrapeErrorsTotalMetric:         eventsScrapeErrorsTotalMetric,
		lastEventsScrapeErrorMetric:           lastEventsScrapeErrorMetric,
		lastEventsScrapeTimestampMetric:       lastEventsScrapeTimestampMetric,
		lastEventsScrapeDurationSecondsMetric: lastEventsScrapeDurationSecondsMetric,
		lastCheckFilter:                       &newTime,
		timeLocation:                          timeLocation,
	}
}

func (c EventsCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()
	errorMetric := float64(0)
	if err := c.reportEventsMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.eventsScrapeErrorsTotalMetric.Inc()
	}
	c.eventsScrapeErrorsTotalMetric.Collect(ch)

	c.eventsScrapesTotalMetric.Inc()
	c.eventsScrapesTotalMetric.Collect(ch)

	c.lastEventsScrapeErrorMetric.Set(errorMetric)
	c.lastEventsScrapeErrorMetric.Collect(ch)

	c.lastEventsScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastEventsScrapeTimestampMetric.Collect(ch)

	c.lastEventsScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastEventsScrapeDurationSecondsMetric.Collect(ch)
}

func (c EventsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.eventsInfoMetric.Describe(ch)
	c.eventsScrapesTotalMetric.Describe(ch)
	c.eventsScrapeErrorsTotalMetric.Describe(ch)
	c.lastEventsScrapeErrorMetric.Describe(ch)
	c.lastEventsScrapeTimestampMetric.Describe(ch)
	c.lastEventsScrapeDurationSecondsMetric.Describe(ch)
}

func (c EventsCollector) reportEventsMetrics(ch chan<- prometheus.Metric) error {
	c.eventsInfoMetric.Reset()
	query := ""

	if c.eventsQuery != "" {
		query = fmt.Sprintf("%s;", c.eventsQuery)
	}

	params := url.Values{}
	params.Set("order-by", "timestamp")
	params.Set("order-direction", "desc")
	params.Set("results-per-page", "10")
	params.Set("q", fmt.Sprintf("%stimestamp>%s", query, *c.lastCheckFilter))

	*c.lastCheckFilter = time.Now().In(c.timeLocation).Format("2006-01-02T15:04:05Z")
	events, err := c.cfClient.ListEventsByQuery(params)
	if err != nil {
		log.Errorf("Error while getting events: %v", err)
		return err
	}

	for _, event := range events {
		c.eventsInfoMetric.WithLabelValues(
			event.GUID,
			event.CreatedAt,
			event.Type,
			event.Actor,
			event.ActorType,
			event.ActorName,
			event.ActorUsername,
			event.Actee,
			event.ActeeType,
			event.ActeeName,
			event.SpaceGUID,
			event.OrganizationGUID,
		).Set(float64(1))
	}

	c.eventsInfoMetric.Collect(ch)
	return nil
}
