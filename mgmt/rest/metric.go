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
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	apiV2 "github.com/intelsdi-x/snap/mgmt/rest/v2"
)

func (s *Server) getMetrics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ver := 0 // 0: get all metrics

	// If we are provided a parameter with the name 'ns' we need to
	// perform a query
	q := r.URL.Query()
	v := q.Get("ver")
	// apiVersion should be "v1" or "v2".
	// If apiVersion != "v2" calls default to V1 functionality
	apiVersion := r.URL.Path[1:3]
	ns_query := q.Get("ns")
	if ns_query != "" {
		ver = 0 // 0: get all versions
		if v != "" {
			var err error
			ver, err = strconv.Atoi(v)
			if err != nil {
				respond(400, rbody.FromError(err), w)
				return
			}
		}
		// strip the leading '/' and split on the remaining '/'
		ns := strings.Split(strings.TrimLeft(ns_query, "/"), "/")
		if ns[len(ns)-1] == "*" {
			ns = ns[:len(ns)-1]
		}

		mts, err := s.mm.FetchMetrics(core.NewNamespace(ns...), ver)
		if err != nil {
			respond(404, rbody.FromError(err), w)
			return
		}
		respondWithMetrics(r.Host, mts, w, apiVersion)
		return
	}

	mts, err := s.mm.MetricCatalog()
	if err != nil {
		respond(500, rbody.FromError(err), w)
		return
	}
	respondWithMetrics(r.Host, mts, w, apiVersion)
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
	// apiVersion should be "v1" or "v2".
	// If apiVersion != "v2" calls default to V1 functionality
	apiVersion := r.URL.Path[1:3]

	if ns[len(ns)-1] == "*" {
		if v == "" {
			ver = 0 //return all metrics
		} else {
			ver, err = strconv.Atoi(v)
			if err != nil {
				respond(400, rbody.FromError(err), w)
				return
			}
		}

		mts, err := s.mm.FetchMetrics(core.NewNamespace(ns[:len(ns)-1]...), ver)
		if err != nil {
			respond(404, rbody.FromError(err), w)
			return
		}
		respondWithMetrics(r.Host, mts, w, apiVersion)
		return
	}

	// If no version was given, get all version that fall at this namespace.
	if v == "" {
		mts, err := s.mm.FetchMetrics(core.NewNamespace(ns...), 0)
		if err != nil {
			respond(404, rbody.FromError(err), w)
			return
		}
		respondWithMetrics(r.Host, mts, w, apiVersion)
		return
	}

	// if an explicit version is given, get that single one.
	ver, err = strconv.Atoi(v)
	if err != nil {
		respond(400, rbody.FromError(err), w)
		return
	}
	mt, err := s.mm.GetMetric(core.NewNamespace(ns...), ver)
	if err != nil {
		respond(404, rbody.FromError(err), w)
		return
	}

	b := &rbody.MetricReturned{}
	dyn, indexes := mt.Namespace().IsDynamic()
	var dynamicElements []rbody.DynamicElement
	if dyn {
		dynamicElements = getDynamicElements(mt.Namespace(), indexes)
	}
	mb := &rbody.Metric{
		Namespace:       mt.Namespace().String(),
		Version:         mt.Version(),
		Dynamic:         dyn,
		DynamicElements: dynamicElements,
		Description:     mt.Description(),
		Unit:            mt.Unit(),
		LastAdvertisedTimestamp: mt.LastAdvertisedTime().Unix(),
		Href: catalogedMetricURI(r.Host, mt),
	}
	policies := rbody.PolicyTableSlice(mt.Policy().RulesAsTable())
	mb.Policy = policies
	b.Metric = mb
	respond(200, b, w)
}

func respondWithMetrics(host string, mts []core.CatalogedMetric, w http.ResponseWriter, version string) {
	if version == "v2" {
		apiV2.RespondWithMetrics(host, mts, w)
		return
	}
	b := rbody.NewMetricsReturned()
	for _, m := range mts {
		policies := rbody.PolicyTableSlice(m.Policy().RulesAsTable())
		dyn, indexes := m.Namespace().IsDynamic()
		var dynamicElements []rbody.DynamicElement
		if dyn {
			dynamicElements = getDynamicElements(m.Namespace(), indexes)
		}
		b = append(b, rbody.Metric{
			Namespace:               m.Namespace().String(),
			Version:                 m.Version(),
			LastAdvertisedTimestamp: m.LastAdvertisedTime().Unix(),
			Description:             m.Description(),
			Dynamic:                 dyn,
			DynamicElements:         dynamicElements,
			Unit:                    m.Unit(),
			Policy:                  policies,
			Href:                    catalogedMetricURI(host, m),
		})
	}
	sort.Sort(b)
	respond(200, b, w)
}

func catalogedMetricURI(host string, mt core.CatalogedMetric) string {
	return fmt.Sprintf("%s://%s/v1/metrics?ns=%s&ver=%d", protocolPrefix, host, url.QueryEscape(mt.Namespace().String()), mt.Version())
}

func getDynamicElements(ns core.Namespace, indexes []int) []rbody.DynamicElement {
	elements := make([]rbody.DynamicElement, 0, len(indexes))
	for _, v := range indexes {
		e := ns.Element(v)
		elements = append(elements, rbody.DynamicElement{
			Index:       v,
			Name:        e.Name,
			Description: e.Description,
		})
	}
	return elements
}
