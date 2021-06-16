package main

import (
	"fmt"
	"log"
	"strconv"

	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"time"
)

func timeSpan() (string, string) {
	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	ThreeDaysAgo := now.AddDate(0, 0, -3).Format("2006-01-02")
	return ThreeDaysAgo, twoDaysAgo
}

var (
	awsDailyUnblendedCosts float64
	billingDate            string
)

type myCollector struct {
	metric *prometheus.Desc
}

func (c *myCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metric
}

func (c *myCollector) Collect(ch chan<- prometheus.Metric) {

	t, _ := time.Parse("2006-01-02", billingDate)
	s := prometheus.NewMetricWithTimestamp(t, prometheus.MustNewConstMetric(c.metric, prometheus.GaugeValue, awsDailyUnblendedCosts))

	ch <- s
}

func main() {

	collector := &myCollector{
		metric: prometheus.NewDesc(
			"daily_unblended_costs",
			"AWS Daily unbleded costs",
			nil,
			nil,
		),
	}
	prometheus.MustRegister(collector)

	go func() {
		for {
			sess, _ := session.NewSession()
			svc := costexplorer.New(sess)

			start, finish := timeSpan()

			input := &costexplorer.GetCostAndUsageInput{
				Granularity: aws.String("DAILY"),
				TimePeriod: &costexplorer.DateInterval{
					Start: aws.String(start),
					End:   aws.String(finish),
				},
				Metrics: []*string{
					aws.String("UNBLENDED_COST"),
				},
			}

			var req *request.Request
			var resp *costexplorer.GetCostAndUsageOutput
			req, resp = svc.GetCostAndUsageRequest(input)

			err := req.Send()
			if err != nil {
				fmt.Println(err)
			}
			billingDate = *resp.ResultsByTime[0].TimePeriod.Start
			awsDailyUnblendedCosts, _ = strconv.ParseFloat(*resp.ResultsByTime[0].Total["UnblendedCost"].Amount, 64)

			time.Sleep(4 * 3600 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":5000", nil))
}
