package main

import (
	"fmt"
	"time"

	"github.com/awootton/knotfreeiot/promutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func main() {
	fmt.Println("hello aa")

	recordMetrics()

	go promutil.StartPromServer()

	

	fmt.Println("back!")
	for {
		time.Sleep(1 * time.Second)
	}
}

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)

			sss := promutil.GetReport()
			fmt.Println(sss)
		}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "atw_test_counter_counted_total",
		Help: "The total number of counts",
	})
)
