// Copyright 2019,2020 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpServe404 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_http_404",
			Help: "Number of 404 main.ServeHTTP.",
		},
	)

	forwardsCount3000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_3000_forwards",
			Help: "Number forwards main.startPublicServer3000.",
		},
	)

	forwardsCount9090 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_9090_forwards",
			Help: "http forwards main.startPublicServer9090.",
		},
	)

	forwardsCount8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_forwards",
			Help: "tcp count main.startPublicServer9090.",
		},
	)
	forwardsDialFail8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_dialfail",
			Help: "tcp dialfail main.startPublicServer9090.",
		},
	)

	forwardsConnectedl8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_connected",
			Help: "tcp conected main.startPublicServer9090.",
		},
	)

	forwardsAcceptl8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_accepted",
			Help: "tcp accepted main.startPublicServer9090.",
		},
	)
)
