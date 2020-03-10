package iot

import "github.com/prometheus/client_golang/prometheus"

var (
	// API1GetStats is
	API1GetStats = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "api1_getstats",
			Help: "http requests.",
		},
	)
	// IotHTTP404 is used in TCPUtil.go
	IotHTTP404 = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "iot_http_404",
			Help: "http 404.",
		},
	)

	// API1PostGurus is
	API1PostGurus = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "api2_post_gurus",
			Help: "http post /api2/set.",
		},
	)

	// API1PostGurusFail is searchable
	API1PostGurusFail = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "api1_post_gurus_fail",
			Help: "http post /api2/set.",
		},
	)

	// TCPNameResolverFail1 is
	TCPNameResolverFail1 = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_fail1",
			Help: "looking for peers.",
		},
	)
	// TCPNameResolverFail2 is
	TCPNameResolverFail2 = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_fail2",
			Help: "looking for peers.",
		},
	)

	// TCPNameResolverConnected is
	TCPNameResolverConnected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_name_resolver_connected",
			Help: "looking for peers.",
		},
	)
	//TCPServerDidntStart is
	TCPServerDidntStart = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_fail1",
			Help: "looking for peers.",
		},
	)
	//TCPServerAcceptError is
	TCPServerAcceptError = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_fail1",
			Help: "looking for peers.",
		},
	)

	//TCPServerConnAccept is
	TCPServerConnAccept = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_cpnn_accept",
			Help: "looking for peers.",
		},
	)

	//TCPServerNewConnection is
	TCPServerNewConnection = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_new_conn",
			Help: "looking for peers.",
		},
	)
	//TCPServerPacketReadError is
	TCPServerPacketReadError = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_cpnn_accept",
			Help: "looking for peers.",
		},
	)

	//TCPServerIotPushEror is
	TCPServerIotPushEror = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcp_server_cpnn_accept",
			Help: "looking for peers.",
		},
	)
)
