/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rest

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"golang.org/x/net/context"

	"github.com/julienschmidt/httprouter"

	"github.com/intelsdi-x/snap/control/rpc"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
)

func (s *Server) getMetrics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	arg := &rpc.EmptyRequest{}
	reply, err := s.mm.MetricCatalog(context.Background(), arg)
	if err != nil {
		respond(500, rbody.FromError(err), w)
		return
	}
	metricResponse(r.Host, rpc.ReplyToMetrics(reply.Metrics), w)
}

func metricResponse(host string, metrics []rpc.Metric, w http.ResponseWriter) {
	b := rbody.NewMetricsReturned()

	for _, met := range metrics {
		// Merge policy with new policy here because metric.ConfigPolicy will have
		// a nil mutex. Merging gives us a usable mutex so RulesAsTable doesn't Panic.
		rt := met.ConfigPolicy.RulesAsTable()
		policies := make([]rbody.PolicyTable, 0, len(rt))
		for _, r := range rt {
			policies = append(policies, rbody.PolicyTable{
				Name:     r.Name,
				Type:     r.Type,
				Default:  r.Default,
				Required: r.Required,
				Minimum:  r.Minimum,
				Maximum:  r.Maximum,
			})
		}
		b = append(b, rbody.Metric{
			Namespace:               core.JoinNamespace(met.Namespace),
			Version:                 met.Version,
			LastAdvertisedTimestamp: met.LastAdvertisedTime.Unix(),
			Policy:                  policies,
			Href:                    catalogedMetricURI(host, met),
		})
	}
	sort.Sort(b)
	// If a single metric use rbody.MetricReturned because that is what the cli expects
	if len(b) == 1 {
		single := &rbody.MetricReturned{}
		single.Metric = &b[0]
		respond(200, single, w)
	} else {
		respond(200, b, w)
	}
}

func (s *Server) getMetricsFromTree(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	namespace := params.ByName("namespace")

	// we land here if the request contains a trailing slash, because it matches the tree
	// lookup URL: /v1/metrics/*namespace.  If the length of the namespace param is 1, we
	// redirect the request to getMetrics.  This results in GET /v1/metrics and
	// GET /v1/metrics/ behaving the same way.
	if len(namespace) <= 1 {
		s.getMetrics(w, r, params)
		return
	}

	ns := parseNamespace(namespace)

	var (
		ver int
		err error
	)
	q := r.URL.Query()
	v := q.Get("ver")

	if ns[len(ns)-1] == "*" {
		if v == "" {
			ver = -1
		} else {
			ver, err = strconv.Atoi(v)
			if err != nil {
				respond(400, rbody.FromError(err), w)
				return
			}
		}
		arg := &rpc.FetchMetricsRequest{
			Namespace: ns[:len(ns)-1],
			Version:   int32(ver),
		}
		reply, err := s.mm.FetchMetrics(context.Background(), arg)
		if err != nil {
			respond(404, rbody.FromError(err), w)
			return
		}
		metricResponse(r.Host, rpc.ReplyToMetrics(reply.Metrics), w)
		return
	}

	// If no version was given, get all that fall at this namespace.
	if v == "" {
		arg := &rpc.GetMetricVersionsRequest{
			Namespace: ns,
		}
		reply, err := s.mm.GetMetricVersions(context.Background(), arg)
		if err != nil {
			respond(404, rbody.FromError(err), w)

			return
		}
		metricResponse(r.Host, rpc.ReplyToMetrics(reply.Metrics), w)
		return
	}

	// if an explicit version is given, get that single one.
	ver, err = strconv.Atoi(v)
	if err != nil {
		respond(400, rbody.FromError(err), w)
		return
	}
	arg := &rpc.FetchMetricsRequest{
		Namespace: ns,
		Version:   int32(ver),
	}
	reply, err := s.mm.GetMetric(context.Background(), arg)
	if err != nil {
		respond(404, rbody.FromError(err), w)
	}
	metricResponse(r.Host, rpc.ReplyToMetrics([]*rpc.MetricReply{reply}), w)
}

func catalogedMetricURI(host string, mt rpc.Metric) string {
	return fmt.Sprintf("%s://%s/v1/metrics%s?ver%d", protocolPrefix, host, core.JoinNamespace(mt.Namespace), mt.Version)
}
