/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2016 Intel Corporation

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

package controlproxy

import (
	"errors"
	"testing"
	"time"

	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
	"github.com/intelsdi-x/snap/grpc/common"
	"github.com/intelsdi-x/snap/grpc/controlproxy/rpc"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	rpcErr = errors.New("RPC ERROR")
)

type mockClient struct {
	RpcErr           bool
	ExpandReply      *rpc.ExpandWildcardsReply
	PublishReply     *rpc.ErrorReply
	ProcessReply     *rpc.ProcessMetricsReply
	CollectReply     *rpc.CollectMetricsReply
	ContentTypeReply *rpc.GetPluginContentTypesReply
	ValidateReply    *rpc.ValidateDepsReply
	SubscribeReply   *rpc.SubscribeDepsReply
	UnsubscribeReply *rpc.SubscribeDepsReply
	MatchReply       *rpc.ExpandWildcardsReply
}

func (mc mockClient) GetPluginContentTypes(ctx context.Context, in *rpc.GetPluginContentTypesRequest, opts ...grpc.CallOption) (*rpc.GetPluginContentTypesReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.ContentTypeReply, nil
}
func (mc mockClient) ExpandWildcards(ctx context.Context, in *rpc.ExpandWildcardsRequest, opts ...grpc.CallOption) (*rpc.ExpandWildcardsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.ExpandReply, nil
}
func (mc mockClient) CollectMetrics(ctx context.Context, in *rpc.CollectMetricsRequest, opts ...grpc.CallOption) (*rpc.CollectMetricsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.CollectReply, nil
}
func (mc mockClient) PublishMetrics(ctx context.Context, in *rpc.PubProcMetricsRequest, opts ...grpc.CallOption) (*rpc.ErrorReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.PublishReply, nil
}
func (mc mockClient) ProcessMetrics(ctx context.Context, in *rpc.PubProcMetricsRequest, opts ...grpc.CallOption) (*rpc.ProcessMetricsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.ProcessReply, nil
}
func (mc mockClient) ValidateDeps(ctx context.Context, in *rpc.ValidateDepsRequest, opts ...grpc.CallOption) (*rpc.ValidateDepsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.ValidateReply, nil
}
func (mc mockClient) SubscribeDeps(ctx context.Context, in *rpc.SubscribeDepsRequest, opts ...grpc.CallOption) (*rpc.SubscribeDepsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.SubscribeReply, nil
}
func (mc mockClient) UnsubscribeDeps(ctx context.Context, in *rpc.SubscribeDepsRequest, opts ...grpc.CallOption) (*rpc.SubscribeDepsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.UnsubscribeReply, nil
}
func (mc mockClient) MatchQueryToNamespaces(ctx context.Context, in *rpc.ExpandWildcardsRequest, opts ...grpc.CallOption) (*rpc.ExpandWildcardsReply, error) {
	if mc.RpcErr {
		return nil, rpcErr
	}
	return mc.MatchReply, nil
}

func TestExpandWildcards(t *testing.T) {
	Convey("RPC client errors", t, func() {
		input := core.NewNamespace("testing", "this")

		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		ns, err := proxy.ExpandWildcards(input)

		Convey("So the error should be passed through", func() {
			So(err.Error(), ShouldResemble, rpcErr.Error())
		})
		Convey("So The namespace ShouldBeNil", func() {
			So(ns, ShouldBeNil)
		})
	})

	Convey("call to Control.ExpandWildcards returns error", t, func() {
		input := core.NewNamespace("testing", "this")
		reply := &rpc.ExpandWildcardsReply{
			Error: &common.SnapError{
				ErrorFields: map[string]string{},
				ErrorString: "Error from control",
			},
		}

		proxy := ControlProxy{Client: mockClient{ExpandReply: reply}}
		ns, err := proxy.ExpandWildcards(input)

		Convey("So the err should be: "+reply.Error.ErrorString, func() {
			So(err.Error(), ShouldResemble, common.ToSnapError(reply.Error).Error())
		})
		Convey("So Namespaces should be nil", func() {
			So(ns, ShouldBeNil)
		})
	})

	Convey("Successful call", t, func() {
		input := core.NewNamespace("testing", "this")
		a := core.NewNamespace("testing", "this")
		b := core.NewNamespace("stuff", "more")
		proto_a := &rpc.ArrString{S: common.ToNamespace(a)}
		proto_b := &rpc.ArrString{S: common.ToNamespace(b)}
		reply := &rpc.ExpandWildcardsReply{
			Error: nil,
			NSS:   []*rpc.ArrString{proto_a, proto_b},
		}

		proxy := ControlProxy{Client: mockClient{ExpandReply: reply}}
		ns, err := proxy.ExpandWildcards(input)

		Convey("so the err Should be nil", func() {
			So(err, ShouldBeNil)
		})
		Convey("So namespaces should resemble:"+a.String()+","+b.String(), func() {
			So(ns, ShouldResemble, []core.Namespace{a, b})
		})

	})
}

func TestPublishMetrics(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		errs := proxy.PublishMetrics("", []byte{}, "fake", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.Publish returns an error", t, func() {
		reply := &rpc.ErrorReply{
			Errors: []string{"errors"},
		}

		proxy := ControlProxy{Client: mockClient{PublishReply: reply}}
		errs := proxy.PublishMetrics("", []byte{}, "fake", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So err should not be nil", func() {
			So(errs, ShouldNotBeNil)
		})

		Convey("So errs should contain 'errors'", func() {
			So(errs[0].Error(), ShouldResemble, "errors")
		})
	})

	Convey("control.Publish does not error", t, func() {
		reply := &rpc.ErrorReply{Errors: []string{}}

		proxy := ControlProxy{Client: mockClient{PublishReply: reply}}
		errs := proxy.PublishMetrics("", []byte{}, "fake", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So publishing should not error", func() {
			So(len(errs), ShouldEqual, 0)
		})
	})
}

func TestProcessMetrics(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		_, _, errs := proxy.ProcessMetrics("", []byte{}, "fake", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.Process returns an error", t, func() {
		reply := &rpc.ProcessMetricsReply{
			ContentType: "bogus",
			Content:     []byte{},
			Errors:      []string{"error in control.Process"},
		}

		proxy := ControlProxy{Client: mockClient{ProcessReply: reply}}
		_, _, errs := proxy.ProcessMetrics("", []byte{}, "", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So errs should not be nil", func() {
			So(errs, ShouldNotBeNil)
		})

		Convey("So len of errs should be 1", func() {
			So(len(errs), ShouldEqual, 1)
		})

		Convey("So that error should have content 'error in control.Process'", func() {
			So(errs[0].Error(), ShouldResemble, "error in control.Process")
		})
	})

	Convey("Control.Process returns successfully", t, func() {
		reply := &rpc.ProcessMetricsReply{
			ContentType: "bogus",
			Content:     []byte{},
			Errors:      []string{},
		}

		proxy := ControlProxy{Client: mockClient{ProcessReply: reply}}
		ct, _, errs := proxy.ProcessMetrics("", []byte{}, "", 1, map[string]ctypes.ConfigValue{}, "")

		Convey("So len of errs should be 0", func() {
			So(len(errs), ShouldEqual, 0)
		})

		Convey("So returned content-type should be 'bogus'", func() {
			So(ct, ShouldResemble, "bogus")
		})
	})
}

func TestCollectMetrics(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		_, errs := proxy.CollectMetrics([]core.Metric{}, time.Now(), "")

		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.CollectMetrics returns an error", t, func() {
		reply := &rpc.CollectMetricsReply{
			Metrics: nil,
			Errors:  []string{"error in collect"},
		}

		proxy := ControlProxy{Client: mockClient{CollectReply: reply}}
		_, errs := proxy.CollectMetrics([]core.Metric{}, time.Now(), "")

		Convey("So len of errs should be 1", func() {
			So(len(errs), ShouldEqual, 1)
		})

		Convey("So error should contain the string 'error in collect'", func() {
			So(errs[0].Error(), ShouldResemble, "error in collect")
		})
	})

	Convey("Control.CollectMetrics returns sucessfully", t, func() {
		reply := &rpc.CollectMetricsReply{
			Metrics: []*common.Metric{&common.Metric{
				Namespace:          common.ToNamespace(core.NewNamespace("testing", "this")),
				Version:            6,
				Tags:               map[string]string{},
				Timestamp:          &common.Time{Sec: time.Now().Unix(), Nsec: int64(time.Now().Nanosecond())},
				LastAdvertisedTime: &common.Time{Sec: time.Now().Unix(), Nsec: int64(time.Now().Nanosecond())},
			}},
			Errors: nil,
		}

		proxy := ControlProxy{Client: mockClient{CollectReply: reply}}
		mts, errs := proxy.CollectMetrics([]core.Metric{}, time.Now(), "")

		Convey("So len of errs should be 0", func() {
			So(len(errs), ShouldEqual, 0)
		})

		Convey("So mts should not be nil", func() {
			So(mts, ShouldNotBeNil)
		})

		Convey("So len of metrics returned should be 1", func() {
			So(len(mts), ShouldEqual, 1)
		})
	})
}

func TestGetPluginContentTypes(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		_, _, err := proxy.GetPluginContentTypes("", core.PluginType(1), 2)

		Convey("So the error should be passed through", func() {
			So(err.Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("control.GetPluginContentTypes returns an error", t, func() {
		reply := &rpc.GetPluginContentTypesReply{
			AcceptedTypes: []string{"accept"},
			ReturnedTypes: []string{"return"},
			Error:         "error",
		}

		proxy := ControlProxy{Client: mockClient{ContentTypeReply: reply}}
		_, _, err := proxy.GetPluginContentTypes("", core.PluginType(1), 2)

		Convey("So err should resemble 'error' ", func() {
			So(err.Error(), ShouldResemble, "error")
		})
	})

	Convey("control.GetPluginContentTypes returns successfully", t, func() {
		reply := &rpc.GetPluginContentTypesReply{
			AcceptedTypes: []string{"accept"},
			ReturnedTypes: []string{"return"},
		}

		proxy := ControlProxy{Client: mockClient{ContentTypeReply: reply}}
		act, rct, err := proxy.GetPluginContentTypes("", core.PluginType(1), 2)

		Convey("So err should be nil", func() {
			So(err, ShouldBeNil)
		})

		Convey("So accepted/returned types should not be nil", func() {
			So(act, ShouldNotBeNil)
			So(rct, ShouldNotBeNil)
		})

		Convey("So accepted should contain 'accept'", func() {
			So(act, ShouldContain, "accept")
		})

		Convey("So returned should contain 'return'", func() {
			So(rct, ShouldContain, "return")
		})
	})
}

func TestValidateDeps(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		errs := proxy.ValidateDeps([]core.Metric{}, []core.SubscribedPlugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldBeGreaterThan, 0)
		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.ValidateDeps returns an error", t, func() {
		reply := &rpc.ValidateDepsReply{
			Errors: []*common.SnapError{&common.SnapError{ErrorFields: map[string]string{}, ErrorString: "test"}},
		}

		proxy := ControlProxy{Client: mockClient{ValidateReply: reply}}
		errs := proxy.ValidateDeps([]core.Metric{}, []core.SubscribedPlugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldEqual, 1)
		Convey("So the error should resemble 'test'", func() {
			So(errs[0].Error(), ShouldResemble, "test")
		})

	})

	Convey("Control.ValidateDeps returns successfully", t, func() {
		reply := &rpc.ValidateDepsReply{}

		proxy := ControlProxy{Client: mockClient{ValidateReply: reply}}
		errs := proxy.ValidateDeps([]core.Metric{}, []core.SubscribedPlugin{})
		Convey("So the there should be no errors", func() {
			So(len(errs), ShouldEqual, 0)
		})

	})
}

func TestSubscribeDeps(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		errs := proxy.SubscribeDeps("", []core.Metric{}, []core.Plugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldBeGreaterThan, 0)
		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.SubscribeDeps returns an error", t, func() {
		reply := &rpc.SubscribeDepsReply{
			Errors: []*common.SnapError{&common.SnapError{ErrorFields: map[string]string{}, ErrorString: "test"}},
		}

		proxy := ControlProxy{Client: mockClient{SubscribeReply: reply}}
		errs := proxy.SubscribeDeps("", []core.Metric{}, []core.Plugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldEqual, 1)
		Convey("So the error should resemble 'test'", func() {
			So(errs[0].Error(), ShouldResemble, "test")
		})

	})

	Convey("Control.SubscribeDeps returns successfully", t, func() {
		reply := &rpc.SubscribeDepsReply{}

		proxy := ControlProxy{Client: mockClient{SubscribeReply: reply}}
		errs := proxy.SubscribeDeps("", []core.Metric{}, []core.Plugin{})
		Convey("So the there should be no errors", func() {
			So(len(errs), ShouldEqual, 0)
		})

	})
}

func TestUnsubscribeDeps(t *testing.T) {
	Convey("RPC client errors", t, func() {
		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		errs := proxy.UnsubscribeDeps("", []core.Metric{}, []core.Plugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldBeGreaterThan, 0)
		Convey("So the error should be passed through", func() {
			So(errs[0].Error(), ShouldResemble, rpcErr.Error())
		})
	})

	Convey("Control.UnsubscribeDeps returns an error", t, func() {
		reply := &rpc.SubscribeDepsReply{
			Errors: []*common.SnapError{&common.SnapError{ErrorFields: map[string]string{}, ErrorString: "test"}},
		}

		proxy := ControlProxy{Client: mockClient{UnsubscribeReply: reply}}
		errs := proxy.UnsubscribeDeps("", []core.Metric{}, []core.Plugin{})
		So(errs, ShouldNotBeNil)
		So(len(errs), ShouldEqual, 1)
		Convey("So the error should resemble 'test'", func() {
			So(errs[0].Error(), ShouldResemble, "test")
		})

	})

	Convey("Control.UnsubscribeDeps returns successfully", t, func() {
		reply := &rpc.SubscribeDepsReply{}

		proxy := ControlProxy{Client: mockClient{UnsubscribeReply: reply}}
		errs := proxy.UnsubscribeDeps("", []core.Metric{}, []core.Plugin{})
		Convey("So the there should be no errors", func() {
			So(len(errs), ShouldEqual, 0)
		})

	})
}

func TestMatchQueryToNamespaces(t *testing.T) {
	Convey("RPC client errors", t, func() {
		input := core.NewNamespace("testing", "this")

		proxy := ControlProxy{Client: mockClient{RpcErr: true}}
		ns, err := proxy.ExpandWildcards(input)

		Convey("So the error should be passed through", func() {
			So(err.Error(), ShouldResemble, rpcErr.Error())
		})
		Convey("So The namespace ShouldBeNil", func() {
			So(ns, ShouldBeNil)
		})
	})

	Convey("call to Control.MatchQueryToNamespaces returns error", t, func() {
		input := core.NewNamespace("testing", "this")
		reply := &rpc.ExpandWildcardsReply{
			Error: &common.SnapError{
				ErrorFields: map[string]string{},
				ErrorString: "Error from control",
			},
		}

		proxy := ControlProxy{Client: mockClient{MatchReply: reply}}
		ns, err := proxy.MatchQueryToNamespaces(input)

		Convey("So the err should be: "+reply.Error.ErrorString, func() {
			So(err.Error(), ShouldResemble, common.ToSnapError(reply.Error).Error())
		})
		Convey("So Namespaces should be nil", func() {
			So(ns, ShouldBeNil)
		})
	})

	Convey("Control.MatchQueryToNamespaces returns successfully", t, func() {
		input := core.NewNamespace("testing", "this")
		a := core.NewNamespace("testing", "this")
		b := core.NewNamespace("stuff", "more")
		proto_a := &rpc.ArrString{S: common.ToNamespace(a)}
		proto_b := &rpc.ArrString{S: common.ToNamespace(b)}
		reply := &rpc.ExpandWildcardsReply{
			Error: nil,
			NSS:   []*rpc.ArrString{proto_a, proto_b},
		}

		proxy := ControlProxy{Client: mockClient{MatchReply: reply}}
		ns, err := proxy.MatchQueryToNamespaces(input)

		Convey("so the err Should be nil", func() {
			So(err, ShouldBeNil)
		})
		Convey("So namespaces should resemble:"+a.String()+","+b.String(), func() {
			So(ns, ShouldResemble, []core.Namespace{a, b})
		})

	})

}
