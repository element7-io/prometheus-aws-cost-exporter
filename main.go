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
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"time"
)

func timeSpan() (string, string) {
	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2).Format("2006-01-02")
	ThreeDaysAgo := now.AddDate(0, 0, -3).Format("2006-01-02")
	return ThreeDaysAgo, twoDaysAgo
}

func recordMetrics() {
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
			awsCosts, _ := strconv.ParseFloat(*resp.ResultsByTime[0].Total["UnblendedCost"].Amount, 64)
			opsAwsCosts.Set(awsCosts)

			time.Sleep(4 * 3600 * time.Second)
		}
	}()
}

var (
	opsAwsCosts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "daily_unblended_costs",
		Help: "AWS Daily unbleded costs",
	})
)

func main() {

	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":5000", nil))
}
