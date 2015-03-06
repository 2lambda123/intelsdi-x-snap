package facter

import (
	"reflect"
	"testing"

	"github.com/intelsdilabs/pulse/control/plugin"
	. "github.com/smartystreets/goconvey/convey"
)

const FACTER_NAME = "intel/facter"

func TestFacterConfigTests(t *testing.T) {
	Convey("Facter plugin constants tests", t, func() {
		Convey("Facter name should resemble intel/facter", func() {
			So(Name, ShouldResemble, FACTER_NAME)
		})
		Convey("Facter type should be plugin.CollectorPluginType", func() {
			So(Type, ShouldEqual, plugin.CollectorPluginType)
		})
	})
}

func TestFacterGetMetrics(t *testing.T) {

	Convey("GetMetricTypes tests", t, func() {
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
	Convey("TestFacterCollect tests", t, func() {
		f := NewFacterPlugin()
		Convey("update ache", func() {
			f.updateCache([]string{"foo"})
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
		Convey("Name should be inte/facter", func() {
			So(meta.Name, ShouldResemble, "intel/facter")
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
