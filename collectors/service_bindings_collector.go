package collectors

import (
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type ServiceBindingsCollector struct {
	namespace                                               string
	environment                                             string
	deployment                                              string
	cfClient                                                *cfclient.Client
	serviceBindingInfoMetric                                *prometheus.GaugeVec
	serviceBindingsScrapesTotalMetric                       prometheus.Counter
	serviceBindingsScrapeErrorsTotalMetric                  prometheus.Counter
	lastServiceBindingsScrapeErrorMetric                    prometheus.Gauge
	lastServiceBindingsScrapeTimestampMetric                prometheus.Gauge
	lastServiceBindingsScrapeDurationSecondsMetric          prometheus.Gauge
	lastServiceBindingsScheduledScrapeDurationSecondsMetric prometheus.Gauge
}

var (
	cachedServiceBindings      = []cfclient.ServiceBinding(nil)
	cachedServiceBindingsError = error(nil)
	cachedServiceBindingsTime  = 0.0
)

func cacheServiceBindingsMetrics(cfClient *cfclient.Client) {
	log.Infof("Scraping...")
	var begun = time.Now()

	serviceBindings, err := cfClient.ListServiceBindings()
	if err != nil {
		log.Errorf("Error while listing routes: %v", err)
	}
	cachedServiceBindings = serviceBindings
	cachedServiceBindingsError = err

	cachedServiceBindingsTime = time.Since(begun).Seconds()
}

func NewServiceBindingsCollector(
	namespace string,
	environment string,
	deployment string,
	cfClient *cfclient.Client,
) *ServiceBindingsCollector {
	schedule(cfClient, cacheServiceBindingsMetrics, 300*time.Second)

	serviceBindingInfoMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "service_binding",
			Name:        "info",
			Help:        "Labeled Cloud Foundry Service Binding information with a constant '1' value.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
		[]string{"service_binding_id", "application_id", "service_instance_id"},
	)

	serviceBindingsScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "service_bindings_scrapes",
			Name:        "total",
			Help:        "Total number of scrapes for Cloud Foundry Service Bindings.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	serviceBindingsScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "service_bindings_scrape_errors",
			Name:        "total",
			Help:        "Total number of scrape error of Cloud Foundry Service Bindings.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastServiceBindingsScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_service_bindings_scrape_error",
			Help:        "Whether the last scrape of Service Bindings metrics from Cloud Foundry resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastServiceBindingsScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_service_bindings_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Service Bindings metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastServiceBindingsScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_service_bindings_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Service Bindings metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	lastServiceBindingsScheduledScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_service_bindings_scheduled_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Service Bindings metrics from Cloud Foundry.",
			ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
		},
	)

	return &ServiceBindingsCollector{
		namespace:                                      namespace,
		environment:                                    environment,
		deployment:                                     deployment,
		cfClient:                                       cfClient,
		serviceBindingInfoMetric:                       serviceBindingInfoMetric,
		serviceBindingsScrapesTotalMetric:              serviceBindingsScrapesTotalMetric,
		serviceBindingsScrapeErrorsTotalMetric:         serviceBindingsScrapeErrorsTotalMetric,
		lastServiceBindingsScrapeErrorMetric:           lastServiceBindingsScrapeErrorMetric,
		lastServiceBindingsScrapeTimestampMetric:       lastServiceBindingsScrapeTimestampMetric,
		lastServiceBindingsScrapeDurationSecondsMetric: lastServiceBindingsScrapeDurationSecondsMetric,
		lastServiceBindingsScheduledScrapeDurationSecondsMetric: lastServiceBindingsScheduledScrapeDurationSecondsMetric,
	}
}

func (c ServiceBindingsCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportServiceBindingsMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.serviceBindingsScrapeErrorsTotalMetric.Inc()
	}
	c.serviceBindingsScrapeErrorsTotalMetric.Collect(ch)

	c.serviceBindingsScrapesTotalMetric.Inc()
	c.serviceBindingsScrapesTotalMetric.Collect(ch)

	c.lastServiceBindingsScrapeErrorMetric.Set(errorMetric)
	c.lastServiceBindingsScrapeErrorMetric.Collect(ch)

	c.lastServiceBindingsScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastServiceBindingsScrapeTimestampMetric.Collect(ch)

	c.lastServiceBindingsScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastServiceBindingsScrapeDurationSecondsMetric.Collect(ch)

	c.lastServiceBindingsScheduledScrapeDurationSecondsMetric.Set(cachedServiceBindingsTime)
	c.lastServiceBindingsScheduledScrapeDurationSecondsMetric.Collect(ch)
}

func (c ServiceBindingsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.serviceBindingInfoMetric.Describe(ch)
	c.serviceBindingsScrapesTotalMetric.Describe(ch)
	c.serviceBindingsScrapeErrorsTotalMetric.Describe(ch)
	c.lastServiceBindingsScrapeErrorMetric.Describe(ch)
	c.lastServiceBindingsScrapeTimestampMetric.Describe(ch)
	c.lastServiceBindingsScrapeDurationSecondsMetric.Describe(ch)
}

func (c ServiceBindingsCollector) reportServiceBindingsMetrics(ch chan<- prometheus.Metric) error {
	c.serviceBindingInfoMetric.Reset()

	serviceBindings := cachedServiceBindings
	err := cachedServiceBindingsError
	if err != nil {
		log.Errorf("Error while listing service bindings: %v", err)
		return err
	}

	for _, serviceBinding := range serviceBindings {
		c.serviceBindingInfoMetric.WithLabelValues(
			serviceBinding.Guid,
			serviceBinding.AppGuid,
			serviceBinding.ServiceInstanceGuid,
		).Set(float64(1))
	}

	c.serviceBindingInfoMetric.Collect(ch)

	return nil
}
