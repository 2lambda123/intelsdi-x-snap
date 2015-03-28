package client

import (
	"encoding/json"
	"encoding/gob"
	"net"
	"net/rpc"
	"time"

	"github.com/intelsdilabs/pulse/control/plugin"
	"github.com/intelsdilabs/pulse/control/plugin/cpolicy"
	"github.com/intelsdilabs/pulse/core"
)

// Native clients use golang net/rpc for communication to a native rpc server.
type PluginNativeClient struct {
	connection *rpc.Client
}

func NewCollectorNativeClient(address string, timeout time.Duration) (PluginCollectorClient, error) {
	// Attempt to dial address error on timeout or problem
	conn, err := net.DialTimeout("tcp", address, timeout)
	// Return nil RPCClient and err if encoutered
	if err != nil {
		return nil, err
	}
	r := rpc.NewClient(conn)
	p := &PluginNativeClient{connection: r}
	return p, nil
}

func NewPublisherNativeClient(address string, timeout time.Duration) (PluginPublisherClient, error) {
	// Attempt to dial address error on timeout or problem
	conn, err := net.DialTimeout("tcp", address, timeout)
	// Return nil RPCClient and err if encoutered
	if err != nil {
		return nil, err
	}
	r := rpc.NewClient(conn)
	p := &PluginNativeClient{connection: r}
	return p, nil
}

func (p *PluginNativeClient) Ping() error {
	a := plugin.PingArgs{}
	b := true
	err := p.connection.Call("SessionState.Ping", a, &b)
	return err
}

func (p *PluginNativeClient) Kill(reason string) error {
	a := plugin.KillArgs{Reason: reason}
	var b bool
	err := p.connection.Call("SessionState.Kill", a, &b)
	return err
}

func (p *PluginNativeClient) CollectMetrics(coreMetricTypes []core.MetricType) ([]core.Metric, error) {
	// Convert core.MetricType slice into plugin.PluginMetricType slice as we have
	// to send structs over RPC
	pluginMetricTypes := make([]plugin.PluginMetricType, len(coreMetricTypes))
	for i, _ := range coreMetricTypes {
		pluginMetricTypes[i] = *plugin.NewPluginMetricType(coreMetricTypes[i].Namespace())
	}

	// TODO return err if mts is empty
	args := plugin.CollectMetricsArgs{PluginMetricTypes: pluginMetricTypes}
	reply := plugin.CollectMetricsReply{}

	err := p.connection.Call("Collector.CollectMetrics", args, &reply)

	retMetrics := make([]core.Metric, len(reply.PluginMetrics))
	for i, _ := range reply.PluginMetrics {
		retMetrics[i] = reply.PluginMetrics[i]
	}
	return retMetrics, err
}

func (p *PluginNativeClient) GetMetricTypes() ([]core.MetricType, error) {
	args := plugin.GetMetricTypesArgs{}
	reply := plugin.GetMetricTypesReply{}

	err := p.connection.Call("Collector.GetMetricTypes", args, &reply)

	retMetricTypes := make([]core.MetricType, len(reply.PluginMetricTypes))
	for i, _ := range reply.PluginMetricTypes {
		retMetricTypes[i] = reply.PluginMetricTypes[i]
	}
	return retMetricTypes, err
}

func (p *PluginNativeClient) GetConfigPolicyTree() (cpolicy.ConfigPolicyTree, error) {
	// Only types that will be transferred as implementations of interface
	// values need to be registered.
	gob.Register(cpolicy.NewPolicyNode())
	gob.Register(&cpolicy.StringRule{})

	args := plugin.GetConfigPolicyTreeArgs{}
	reply := plugin.GetConfigPolicyTreeReply{PolicyTree: *cpolicy.NewTree()}
	err := p.connection.Call("Collector.GetConfigPolicyTree", args, &reply)
	if err != nil {
		return cpolicy.ConfigPolicyTree{}, err
	}

	return reply.PolicyTree, nil
}

func (p *PluginNativeClient) PublishMetrics(metrics []core.Metric) error {
	reply := new(plugin.PublishReply)

	pluginMetrics := make([]plugin.PluginMetric, len(metrics))
	for i, _ := range metrics {
		pluginMetrics[i] = plugin.PluginMetric{
			Namespace_: metrics[i].Namespace(),
			Data_:      metrics[i].Data()}
	}

	data, err := json.Marshal(pluginMetrics)
	if err != nil {
		return err
	}
	pubArgs := plugin.PublishArgs{Data: data}
	err = p.connection.Call("Publisher.Publish", pubArgs, reply)
	return err
}