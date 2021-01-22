package main

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-ping/ping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/urfave/cli/v2"
)

var (
	app            *cli.App
	pushGatewayURL string
)

const (
	appName = "pinger"
)

func init() {
	pushGatewayURL = os.Getenv("PROMETHEUS_PUSHGATEWAY_URL")

	app = cli.NewApp()
	app.Name = appName
	app.Usage = "The core service."
	app.Action = func(c *cli.Context) error {
		pingInterval := c.Args().Get(0)
		pushInterval := c.Args().Get(1)
		return start(pingInterval, pushInterval)
	}
}

func run() (err error) {
	err = app.Run(os.Args)

	return
}

func start(pingInterval string, pushInterval string) error {
	errC := make(chan error)

	numericRE := regexp.MustCompile("[0-9]+")

	getNumeric := numericRE.FindAllString(pingInterval, -1)
	if len(getNumeric) <= 0 {
		panic("Incorrect parameters")
	}

	interval, _ := strconv.Atoi(getNumeric[0])
	var pingInt *time.Ticker
	if strings.Contains(pingInterval, "m") {
		pingInt = time.NewTicker(time.Duration(interval) * time.Minute)
	} else if strings.Contains(pingInterval, "s") {
		pingInt = time.NewTicker(time.Duration(interval) * time.Second)
	} else {
		panic("Incorrect parameters")
	}

	getNumeric = numericRE.FindAllString(pushInterval, -1)
	if len(getNumeric) <= 0 {
		panic("Incorrect parameters")
	}

	interval, _ = strconv.Atoi(getNumeric[0])
	var pushInt *time.Ticker
	if strings.Contains(pushInterval, "m") {
		pushInt = time.NewTicker(time.Duration(interval) * time.Minute)
	} else if strings.Contains(pushInterval, "s") {
		pushInt = time.NewTicker(time.Duration(interval) * time.Second)
	} else {
		panic("Incorrect parameters")
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: os.Getenv("PROMETHEUS_NAMESPACE"),
		Name:      "round_trip_time",
		Help:      "Round-Trip Time value, generated every (defined) cycle once this service running.",
	})
	if err := prometheus.Register(gauge); err != nil && err.Error() != "duplicate metrics collector registration attempted" {
		panic(err)
	}

	go func() {
		var (
			pingStats *ping.Statistics
			avgRTT    float64
			sum       int64 = 0
			divider   int64 = 0
			targetURL       = os.Getenv("TARGET_URL")
		)

		for {
			select {
			case <-pushInt.C:
				avgRTT = float64(sum) / float64(divider)
				gauge.Set(avgRTT)
				push.New(pushGatewayURL, appName).Collector(gauge).Push()
				sum = 0
				divider = 0
			case <-pingInt.C:
				pingr, _ := ping.NewPinger(targetURL)
				pingr.SetPrivileged(true)
				pingr.Count = 1
				if err := pingr.Run(); err != nil {
					panic(err)
				}

				pingStats = pingr.Statistics()
				sum += int64(pingStats.MaxRtt / time.Millisecond)
				divider++
			}
		}
	}()

	return <-errC
}
