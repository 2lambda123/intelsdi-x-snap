package plugin

import (
	"errors"
	"fmt"

	"github.com/intelsdi-x/pulse/control/plugin/cpolicy"
	"github.com/intelsdi-x/pulse/core/ctypes"
)

type PublishArgs struct {
	//PluginMetrics []PluginMetric
	ContentType string
	Content     []byte
	Config      map[string]ctypes.ConfigValue
}

type PublishReply struct {
}

type publisherPluginProxy struct {
	Plugin  PublisherPlugin
	Session Session
}

type GetConfigPolicyNodeArgs struct{}

type GetConfigPolicyNodeReply struct {
	PolicyNode cpolicy.ConfigPolicyNode
}

func (p *publisherPluginProxy) GetConfigPolicyNode(args GetConfigPolicyNodeArgs, reply *GetConfigPolicyNodeReply) error {
	defer catchPluginPanic(p.Session.Logger())

	p.Session.Logger().Println("GetConfigPolicyNode called")
	p.Session.ResetHeartbeat()

	reply.PolicyNode = p.Plugin.GetConfigPolicyNode()

	return nil
}

func (p *publisherPluginProxy) Publish(args PublishArgs, reply *PublishReply) error {
	defer catchPluginPanic(p.Session.Logger())
	p.Session.ResetHeartbeat()
	err := p.Plugin.PublishType(args.ContentType, args.Content, args.Config)
	if err != nil {
		return errors.New(fmt.Sprintf("Publish call error: %v", err.Error()))
	}
	return nil
}
