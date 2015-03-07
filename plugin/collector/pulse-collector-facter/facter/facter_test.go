package facter

import (
	"reflect"
	"testing"
	"time"

	"github.com/intelsdilabs/pulse/control/plugin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetFacts(t *testing.T) {
	Convey("getFacts ", t, func() {

		Convey("time outs", func() {
			_, _, err := getFacts([]string{}, 0*time.Second)
			So(err, ShouldNotBeNil)
		})

		Convey("returns something withing given time", func() {
			start := time.Now()
			// 4 seconds because default time for goconvey
			facts, when, err := getFacts([]string{}, 4*time.Second)
			So(err, ShouldBeNil)
			So(facts, ShouldNotBeEmpty)
			So(*when, ShouldHappenBetween, start, time.Now())
		})

	})
}

// TODO:
func TODOTestCacheUpdate(t *testing.T) {

	Convey("Facter cache update works", t, func() {

		f := NewFacterPlugin()

		SkipConvey("empty for start", func() {
			So(f.cache, ShouldBeEmpty)
		})

		Convey("updated after first all update", func() {
			err := f.updateCacheAll()
			Printf("\n%v", err)
			So(err, ShouldBeNil)
			So(f.cache, ShouldNotBeEmpty)
		})

	})

	SkipConvey("cache synchronization", t, func() {
		f := NewFacterPlugin()
		Convey("if not synchronized cache is empty", func() {
			f.updateCache([]string{})
			Println("bam", "bam3")
		})

		Convey("cache after first update", func() {

		})

		Convey("cache which needs be resynchronized", func() {

		})

		Convey("asked for no existing fact", func() {

		})

	})
}

func XTestFacterGetMetrics(t *testing.T) {

	// TODO:not implemented! - fullfill GetMetricTypes
	SkipConvey("GetMetricTypes tests", t, func() {

		facter := NewFacterPlugin()
		var pluginArgs plugin.GetMetricTypesArgs
		var reply plugin.GetMetricTypesReply
		Convey("GetMetricsTypes returns no error", func() {
			err := facter.GetMetricTypes(pluginArgs, &reply)
			So(err, ShouldBeNil)
			Convey("metricTypesReply should contain more than zero metrics", func() {
				So(len(reply.MetricTypes), ShouldBeGreaterThan, 0)
			})
			Convey("metricTypesReply contains metric namespace \"intel/facter/kernel\"", func() {
				expectedTimestamp := reply.MetricTypes[0].LastAdvertisedTimestamp()
				expectedNamespace := []string{"intel", "facter", "kernel"}
				expectedMetricType := plugin.NewMetricType(expectedNamespace, expectedTimestamp)
				//					Printf("\n expected: %v\n", expectedMetricType)
				success := false
				for idx, elem := range reply.MetricTypes {
					if reflect.DeepEqual(expectedMetricType, elem) {
						So(reply.MetricTypes[idx], ShouldResemble, expectedMetricType)
						success = true
						break
					}
				}
				if !success {
					// ShouldContain compares through pointers - SO THIS WILL FAIL
					So(reply.MetricTypes, ShouldContain, expectedMetricType)
				}
			})
		})
	})
}

func TestFacterCollect(t *testing.T) {
	// TODO: time outs after 5 seconds because of goroutine
	SkipConvey("TestFacterCollect tests", t, func() {

		f := NewFacterPlugin()
		Convey("update ache", func() {
			f.synchronizeCache([]string{"foo"})
		})

		Convey("Collect returns nil", func() {
			facter := NewFacterPlugin()
			var pluginArgs plugin.CollectorArgs
			var reply plugin.CollectorReply
			So(facter.Collect(pluginArgs, &reply), ShouldBeNil)
		})
	})
}

func TestFacterPluginMeta(t *testing.T) {
	Convey("PluginMeta tests", t, func() {
		meta := Meta()
		Convey("Meta is not nil", func() {
			So(meta, ShouldNotBeNil)
		})
		Convey("Name should be Intel Facter Plugin (c) 2015 Intel Corporation", func() {
			So(meta.Name, ShouldResemble, "Intel Facter Plugin (c) 2015 Intel Corporation")
		})
		Convey("Version should be 1", func() {
			So(meta.Version, ShouldEqual, 1)
		})
		Convey("Type should be plugin.CollectorPluginType", func() {
			So(meta.Type, ShouldResemble, plugin.CollectorPluginType)
		})
	})
}

func TestFacterConfigPolicy(t *testing.T) {
	Convey("TestFacterConfigPolicy tests", t, func() {
		Convey("TestFacterConfigPolicy returns proper object", func() {
			pluginPolicy := new(plugin.ConfigPolicy)
			So(ConfigPolicy(), ShouldResemble, pluginPolicy)
		})
	})
}
