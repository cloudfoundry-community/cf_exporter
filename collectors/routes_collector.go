package collectors

import (
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type RoutesCollector struct {
	namespace                                      string
	environment                                    string
	deployment                                     string
	cfClient                                       *cfclient.Client
	routeInfoMetric                                *prometheus.GaugeVec
	routesScrapesTotalMetric                       prometheus.Counter
	routesScrapeErrorsTotalMetric                  prometheus.Counter
	lastRoutesScrapeErrorMetric                    prometheus.Gauge
	lastRoutesScrapeTimestampMetric                prometheus.Gauge
	lastRoutesScrapeDurationSecondsMetric          prometheus.Gauge
	lastRoutesScheduledScrapeDurationSecondsMetric prometheus.Gauge
}

var (
	cachedRoute      = []cfclient.Route(nil)
	cachedRouteError = error(nil)
	cachedRouteTime  = 0.0
)

func cacheRouteMetrics(cfClient *cfclient.Client) {
	log.Infof("Scraping...")
	var begun = time.Now()

	routes, err := cfClient.ListRoutes()
	if err != nil {
		log.Errorf("Error while listing routes: %v", err)
	}
	cachedRoute = routes
	cachedRouteError = err

	cachedRouteTime = time.Since(begun).Seconds()
}

func NewRoutesCollector(
	namespace string,
	environment string,
	deployment string,
	cfClient *cfclient.Client,
) *RoutesCollector {
	schedule(cfClient, cacheRouteMetrics, 300*time.Second)

	routeInfoMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "route",
			Name:        "info",
			Help:        "Labeled Cloud Foundry Route information with a constant '1' value.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
		[]string{"route_id", "route_host", "route_path", "domain_id", "space_id", "service_instance_id"},
	)

	routesScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "routes_scrapes",
			Name:        "total",
			Help:        "Total number of scrapes for Cloud Foundry Routes.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	routesScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "routes_scrape_errors",
			Name:        "total",
			Help:        "Total number of scrape error of Cloud Foundry Routes.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastRoutesScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_routes_scrape_error",
			Help:        "Whether the last scrape of Routes metrics from Cloud Foundry resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastRoutesScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_routes_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Routes metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastRoutesScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_routes_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Routes metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastRoutesScheduledScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_routes_scheduled_scrape_duration_seconds",
			Help:        "Duration of the last scheduled scrape of Routes metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	return &RoutesCollector{
		namespace:                             namespace,
		environment:                           environment,
		deployment:                            deployment,
		cfClient:                              cfClient,
		routeInfoMetric:                       routeInfoMetric,
		routesScrapesTotalMetric:              routesScrapesTotalMetric,
		routesScrapeErrorsTotalMetric:         routesScrapeErrorsTotalMetric,
		lastRoutesScrapeErrorMetric:           lastRoutesScrapeErrorMetric,
		lastRoutesScrapeTimestampMetric:       lastRoutesScrapeTimestampMetric,
		lastRoutesScrapeDurationSecondsMetric: lastRoutesScrapeDurationSecondsMetric,
		lastRoutesScheduledScrapeDurationSecondsMetric: lastRoutesScheduledScrapeDurationSecondsMetric,
	}
}

func (c RoutesCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportRoutesMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.routesScrapeErrorsTotalMetric.Inc()
	}
	c.routesScrapeErrorsTotalMetric.Collect(ch)

	c.routesScrapesTotalMetric.Inc()
	c.routesScrapesTotalMetric.Collect(ch)

	c.lastRoutesScrapeErrorMetric.Set(errorMetric)
	c.lastRoutesScrapeErrorMetric.Collect(ch)

	c.lastRoutesScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastRoutesScrapeTimestampMetric.Collect(ch)

	c.lastRoutesScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastRoutesScrapeDurationSecondsMetric.Collect(ch)

	c.lastRoutesScheduledScrapeDurationSecondsMetric.Set(cachedRouteTime)
	c.lastRoutesScheduledScrapeDurationSecondsMetric.Collect(ch)
}

func (c RoutesCollector) Describe(ch chan<- *prometheus.Desc) {
	c.routeInfoMetric.Describe(ch)
	c.routesScrapesTotalMetric.Describe(ch)
	c.routesScrapeErrorsTotalMetric.Describe(ch)
	c.lastRoutesScrapeErrorMetric.Describe(ch)
	c.lastRoutesScrapeTimestampMetric.Describe(ch)
	c.lastRoutesScrapeDurationSecondsMetric.Describe(ch)
}

func (c RoutesCollector) reportRoutesMetrics(ch chan<- prometheus.Metric) error {
	c.routeInfoMetric.Reset()

	routes := cachedRoute
	err := cachedRouteError
	if err != nil {
		return err
	}

	for _, route := range routes {
		c.routeInfoMetric.WithLabelValues(
			route.Guid,
			route.Host,
			route.Path,
			route.DomainGuid,
			route.SpaceGuid,
			route.ServiceInstanceGuid,
		).Set(float64(1))
	}

	c.routeInfoMetric.Collect(ch)

	return nil
}
