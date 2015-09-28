/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2015 Intel Coporation

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

package dummy

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/intelsdi-x/pulse/control/plugin"
	"github.com/intelsdi-x/pulse/control/plugin/cpolicy"
)

const (
	// Name of plugin
	Name = "dummy2"
	// Version of plugin
	Version = 2
	// Type of plugin
	Type = plugin.CollectorPluginType
)

// Dummy collector implementation used for testing
type Dummy struct {
}

//Random number generator
func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

// CollectMetrics collects metrics for testing
func (f *Dummy) CollectMetrics(mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	for _, p := range mts {
		log.Println("collecting", p)
	}
	rand.Seed(time.Now().UTC().UnixNano())
	for i, _ := range mts {
		data := randInt(65, 90)
		mts[i].Data_ = data
		mts[i].Source_, _ = os.Hostname()
		mts[i].Timestamp_ = time.Now()
	}
	return mts, nil
}

//GetMetricTypes returns metric types for testing
func (f *Dummy) GetMetricTypes() ([]plugin.PluginMetricType, error) {
	m1 := &plugin.PluginMetricType{Namespace_: []string{"intel", "dummy", "foo"}}
	m2 := &plugin.PluginMetricType{Namespace_: []string{"intel", "dummy", "bar"}}
	return []plugin.PluginMetricType{*m1, *m2}, nil
}

//GetConfigPolicy returns a ConfigPolicy for testing
func (f *Dummy) GetConfigPolicy() (cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("name", false, "bob")
	rule2, _ := cpolicy.NewStringRule("password", true)
	p := cpolicy.NewPolicyNode()
	p.Add(rule)
	p.Add(rule2)
	c.Add([]string{"intel", "dummy", "foo"}, p)
	return *c, nil
}

//Meta returns meta data for testing
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(Name, Version, Type, []string{plugin.PulseGOBContentType}, []string{plugin.PulseGOBContentType})
}
