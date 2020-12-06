package collectors_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	. "github.com/bosh-prometheus/cf_exporter/collectors"
	. "github.com/bosh-prometheus/cf_exporter/utils/test_matchers"
)

func init() {
	log.Base().SetLevel("fatal")
}

var _ = Describe("EventsCollectors", func() {
	var (
		err      error
		server   *ghttp.Server
		cfClient *cfclient.Client

		eventsInfoMetric                      *prometheus.GaugeVec
		eventsScrapesTotalMetric              prometheus.Counter
		eventsScrapeErrorsTotalMetric         prometheus.Counter
		lastEventsScrapeErrorMetric           prometheus.Gauge
		lastEventsScrapeTimestampMetric       prometheus.Gauge
		lastEventsScrapeDurationSecondsMetric prometheus.Gauge

		namespace   = "test_namespace"
		environment = "test_environment"
		deployment  = "test_deployment"

		eventId1        = "fake_event_1"
		createTime1     = "2019-05-11T07:33:45Z"
		type1           = "audit.route.delete-request"
		actor1          = "fake_actor_1"
		actorType1      = "user"
		actorName1      = "fake_user_1"
		actorUsername1  = "fake_username_1"
		actee1          = "fake_actee_1"
		acteeType1      = "route"
		acteeName1      = "fake_actee_1"
		spaceId1        = "fake_space_id_1"
		organizationId1 = "fake_organization_id_1"

		eventId2        = "fake_event_2"
		createTime2     = "2019-05-22T07:33:45Z"
		type2           = "audit.app.update"
		actor2          = "fake_actor_2"
		actorType2      = "user"
		actorName2      = "fake_user_2"
		actorUsername2  = "fake_username_2"
		actee2          = "fake_actee_2"
		acteeType2      = "app"
		acteeName2      = "fake_actee_2"
		spaceId2        = "fake_space_id_2"
		organizationId2 = "fake_organization_id_2"

		eventsCollector *EventsCollector
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v2/info"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			),
		)

		cfClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress: server.URL(),
			Token:      "fake-token",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(server.ReceivedRequests())).To(Equal(1))

		eventsInfoMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "events",
				Name:        "info",
				Help:        "Labeled Cloud Foundry Events information with a constant '1' value.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
			[]string{"event_id","create_time","type", "actor", "actor_type", "actor_name", "actor_username", "actee", "actee_type", "actee_name", "space_id", "organization_id"},
		)
		eventsInfoMetric.WithLabelValues(eventId1,createTime1,type1, actor1, actorType1, actorName1, actorUsername1, actee1, acteeType1, acteeName1, spaceId1, organizationId1).Set(1)
		eventsInfoMetric.WithLabelValues(eventId2,createTime2,type2, actor2, actorType2, actorName2, actorUsername2, actee2, acteeType2, acteeName2, spaceId2, organizationId2).Set(1)

		eventsScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "events_scrapes",
				Name:        "total",
				Help:        "Total number of scrapes for Cloud Foundry Events.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)
		eventsScrapesTotalMetric.Inc()

		eventsScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "events_scrape_errors",
				Name:        "total",
				Help:        "Total number of scrape errors of Cloud Foundry Events.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastEventsScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_events_scrape_error",
				Help:        "Whether the last scrape of Event metrics from Cloud Foundry resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastEventsScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_events_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Event metrics from Cloud Foundry.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)

		lastEventsScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_events_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Events metrics from Cloud Foundry.",
				ConstLabels: prometheus.Labels{"environment": environment, "deployment": deployment},
			},
		)
	})

	JustBeforeEach(func() {
		eventsCollector = NewEventsCollector(namespace, environment, deployment, cfClient, "")
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Describe", func() {
		var (
			descriptions chan *prometheus.Desc
		)

		BeforeEach(func() {
			descriptions = make(chan *prometheus.Desc)
		})

		JustBeforeEach(func() {
			go eventsCollector.Describe(descriptions)
		})

		It("returns a events_info metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(eventsInfoMetric.WithLabelValues(
				type1,
				actor1,
				actorType1,
				actorName1,
				actorUsername1,
				actee1,
				acteeType1,
				acteeName1,
				spaceId1,
				organizationId1,
			).Desc())))
		})

		It("returns a events_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(eventsScrapesTotalMetric.Desc())))
		})

		It("returns a events_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(eventsScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_events_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastEventsScrapeErrorMetric.Desc())))
		})

		It("returns a last_events_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastEventsScrapeTimestampMetric.Desc())))
		})

		It("returns a last_events_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastEventsScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode     int
			eventsResponse cfclient.EventsResponse
			metrics        chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK

			eventsResponse = cfclient.EventsResponse{
				Resources: []cfclient.EventResource{
					cfclient.EventResource{
						Meta: cfclient.Meta{
							Guid: eventId1,
						},
						Entity: cfclient.Event{
							GUID:             eventId1,
							CreatedAt:        createTime1,
							Type:             type1,
							Actor:            actor1,
							ActorType:        actorType1,
							ActorName:        actorName1,
							ActorUsername:    actorUsername1,
							Actee:            actee1,
							ActeeType:        acteeType1,
							ActeeName:        acteeName1,
							SpaceGUID:        spaceId1,
							OrganizationGUID: organizationId1,
						},
					},
					cfclient.EventResource{
						Meta: cfclient.Meta{
							Guid: eventId2,
						},
						Entity: cfclient.Event{
							GUID:             eventId2,
							CreatedAt:        createTime2,
							Type:             type2,
							Actor:            actor2,
							ActorType:        actorType2,
							ActorName:        actorName2,
							ActorUsername:    actorUsername2,
							Actee:            actee2,
							ActeeType:        acteeType2,
							ActeeName:        acteeName2,
							SpaceGUID:        spaceId2,
							OrganizationGUID: organizationId2,
						},
					},
				},
			}

			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v2/events"),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &eventsResponse),
				),
			)

			go eventsCollector.Collect(metrics)
		})

		It("returns an events_info metric for event 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(eventsInfoMetric.WithLabelValues(
				type1,
				actor1,
				actorType1,
				actorName1,
				actorUsername1,
				actee1,
				acteeType1,
				acteeName1,
				spaceId1,
				organizationId1,
			))))
		})

		It("returns an events_info metric for event 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(eventsInfoMetric.WithLabelValues(
				type2,
				actor2,
				actorType2,
				actorName2,
				actorUsername2,
				actee2,
				acteeType2,
				acteeName2,
				spaceId2,
				organizationId2,
			))))
		})

		It("returns an events_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(eventsScrapesTotalMetric)))
		})

		It("returns an events_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(eventsScrapeErrorsTotalMetric)))
		})

		It("returns a last_events_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastEventsScrapeErrorMetric)))
		})

		Context("when it fails to list the events", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError

				eventsScrapeErrorsTotalMetric.Inc()
				lastEventsScrapeErrorMetric.Set(1)
			})

			It("returns an events_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(eventsScrapeErrorsTotalMetric)))
			})

			It("returns a last_events_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastEventsScrapeErrorMetric)))
			})
		})
	})
})
