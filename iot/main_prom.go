// Copyright 2019,2020,2021 Alan Tracey Wootton
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

package iot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPServe404 is
	HTTPServe404 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_http_404",
			Help: "Number of 404 main.ServeHTTP.",
		},
	)
	// ForwardsCount3100 is
	ForwardsCount3100 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_3100_forwards",
			Help: "Number forwards main.startPublicServer3100.",
		},
	)
	// ForwardsCount9090 is for main.go
	ForwardsCount9090 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_9090_forwards",
			Help: "http forwards main.startPublicServer9090.",
		},
	)
	// ForwardsCount8000 is
	ForwardsCount8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_forwards",
			Help: "tcp count main.startPublicServer9090.",
		},
	)
	// ForwardsDialFail8000 is
	ForwardsDialFail8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_dialfail",
			Help: "tcp dialfail main.startPublicServer9090.",
		},
	)
	// ForwardsConnectedl8000 is
	ForwardsConnectedl8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_connected",
			Help: "tcp conected main.startPublicServer9090.",
		},
	)
	// ForwardsAcceptl8000 is
	ForwardsAcceptl8000 = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_8000_accepted",
			Help: "tcp accepted main.startPublicServer9090.",
		},
	)
	// BadTokenRequests is
	BadTokenRequests = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "main_bad_token_requests",
			Help: "Token requests with flaws.",
		},
	)
)
