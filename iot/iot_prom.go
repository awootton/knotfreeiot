package iot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namesAdded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_names_added",
		Help: "The total number of subscriptions requests",
	})
	// TopicsAdded is
	TopicsAdded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_topics_added",
		Help: "The total number new topics/subscriptions] added",
	})

	topicsRemoved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_topics_removed",
		Help: "The total number new topics/subscriptions] deleted",
	})

	missedPushes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_missed_pushes",
		Help: "The total number of publish to empty topic",
	})

	sentMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_sent_messages",
		Help: "The total number of messages sent down",
	})

	fatalMessups = promauto.NewCounter(prometheus.CounterOpts{
		Name: "look_fatal_messages",
		Help: "The total number garbage messages",
	})

	// API1GetStats is
	API1GetStats = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "api1_getstats",
			Help: "http requests.",
		},
	)
	// IotHTTP404 is used in TCPUtil.go
	IotHTTP404 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "iot_http_404",
			Help: "http 404.",
		},
	)

	// API1PostGurus is
	API1PostGurus = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "api2_post_gurus",
			Help: "http post /api2/set.",
		},
	)

	// API1PostGurusFail is searchable
	API1PostGurusFail = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "api1_post_gurus_fail",
			Help: "http post /api2/set.",
		},
	)

	// TCPNameResolverFail1 is
	TCPNameResolverFail1 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_fail1",
			Help: "failed to resolve address of guru.",
		},
	)
	// TCPNameResolverFail2 is
	TCPNameResolverFail2 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_fail2",
			Help: "dial timeout looking for gurus.",
		},
	)

	// TCPNameResolverConnected is
	TCPNameResolverConnected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_connected",
			Help: "normal connect happened.",
		},
	)
	//TCPServerDidntStart is
	TCPServerDidntStart = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_fail1",
			Help: "packet server listen fail.",
		},
	)
	//TCPServerAcceptError is
	TCPServerAcceptError = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_fail2",
			Help: "packet server acceptor fail.",
		},
	)

	//TCPServerConnAccept is
	TCPServerConnAccept = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_conn_accept",
			Help: "normal packer server connection.",
		},
	)

	//TCPServerNewConnection is
	TCPServerNewConnection = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_new_conn",
			Help: "normal packer server connection.",
		},
	)
	//TCPServerPacketReadError is
	TCPServerPacketReadError = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_packet_read_error",
			Help: "packets.ReadPacket error.",
		},
	)

	//TCPServerIotPushEror is
	TCPServerIotPushEror = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_packet_push_error",
			Help: "Push error.",
		},
	)

	//topicsTotal is
	// topicsTotal = promauto.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: "topics_total",
	// 		Help: "Total topic subscribed.",
	// 	},
	// )

	//connectionsTotal is
	connectionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "connections_total",
			Help: "Total subscriptions.",
		},
	)

	//qFullness is normally 0
	// qFullness = promauto.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: "q_percent_total",
	// 		Help: "Usage of queues.",
	// 	},
	// )

	heartbeatLookerDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "heartbeat_looker_seconds",
		Help:    "Histogram for the LookupTableStruct Heartbeat",
		Buckets: prometheus.LinearBuckets(0.01, 0.01, 10),
	})

	heartbeatContactsDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "heartbeat_contacts_seconds",
		Help:    "Histogram for the Executive Heartbeat",
		Buckets: prometheus.LinearBuckets(0.01, 0.01, 10),
	})
)
