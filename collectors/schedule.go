package collectors

import (
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

func schedule(cfClient *cfclient.Client, x func(a *cfclient.Client), interval time.Duration) *time.Ticker {
	x(cfClient)
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			x(cfClient)
		}
	}()
	return ticker
}

func (c ApplicationsCollector) appSchedule(interval time.Duration) *time.Ticker {
	appErrorCache = c.getApplicationMetrics()
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			appErrorCache = c.getApplicationMetrics()
		}
	}()
	return ticker
}
