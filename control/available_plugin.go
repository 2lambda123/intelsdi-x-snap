package control

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intelsdilabs/gomit"
	"github.com/intelsdilabs/pulse/control/plugin"
	"github.com/intelsdilabs/pulse/control/plugin/client"
	"github.com/intelsdilabs/pulse/control/routing"
	"github.com/intelsdilabs/pulse/core/control_event"
	"github.com/intelsdilabs/pulse/pkg/logger"
)

const (
	DefaultClientTimeout           = time.Second * 3
	DefaultHealthCheckTimeout      = time.Second * 1
	DefaultHealthCheckFailureLimit = 3
)

type availablePluginState int

// availablePlugin represents a plugin running and available to execute calls against
type availablePlugin struct {
	Name    string
	Key     string
	Type    plugin.PluginType
	Version int
	Client  client.PluginClient
	Index   int

	id                 int
	hitCount           int
	lastHitTime        time.Time
	eventManager       *gomit.EventController
	failedHealthChecks int
	healthChan         chan error
}

// newAvailablePlugin returns an availablePlugin with information from a
// plugin.Response
func newAvailablePlugin(resp *plugin.Response, id int) (*availablePlugin, error) {
	ap := &availablePlugin{
		Name:    resp.Meta.Name,
		Version: resp.Meta.Version,
		Type:    resp.Type,

		eventManager: new(gomit.EventController),
		healthChan:   make(chan error, 1),
		lastHitTime:  time.Now(),
		id:           id,
	}

	// Create RPC Client
	switch resp.Type {
	case plugin.CollectorPluginType:
		c, e := client.NewCollectorNativeClient(resp.ListenAddress, DefaultClientTimeout)
		ap.Client = c
		if e != nil {
			return nil, errors.New("error while creating client connection: " + e.Error())
		}
	case plugin.PublisherPluginType:
		c, e := client.NewPublisherNativeClient(resp.ListenAddress, DefaultClientTimeout)
		ap.Client = c
		if e != nil {
			return nil, errors.New("error while creating client connection: " + e.Error())
		}
	default:
		return nil, errors.New("Cannot create a client for a plugin of the type: " + resp.Type.String())
	}

	ap.makeKey()
	return ap, nil
}

func (a *availablePlugin) Id() int {
	return a.id
}

func (a *availablePlugin) String() string {
	return fmt.Sprintf("%s:v%d:id%d", a.Name, a.Version, a.id)
}

// Stop halts a running availablePlugin
func (ap *availablePlugin) Stop(r string) error {
	logger.Debug("availableplugin", fmt.Sprintf(ap.Name, ap.Version))
	return ap.Client.Kill(r)
}

// CheckHealth checks the health of a plugin and updates
// ap.failedHealthChecks
func (ap *availablePlugin) CheckHealth() {
	go func() {
		ap.healthChan <- ap.Client.Ping()
	}()
	select {
	case err := <-ap.healthChan:
		if err == nil {
			logger.Debugf("healthcheck", "ok (%s)", ap.String())
			ap.failedHealthChecks = 0
		} else {
			ap.healthCheckFailed()
		}
	case <-time.After(time.Second * 1):
		ap.healthCheckFailed()
	}
}

// healthCheckFailed increments ap.failedHealthChecks and emits a DisabledPluginEvent
// and a HealthCheckFailedEvent
func (ap *availablePlugin) healthCheckFailed() {
	logger.Debugf("heartbeat", "missed (%s)", ap.String())
	ap.failedHealthChecks++
	if ap.failedHealthChecks >= DefaultHealthCheckFailureLimit {
		logger.Debugf("hearbeat", "failed (%s)", ap.String())
		pde := &control_event.DisabledPluginEvent{
			Name:    ap.Name,
			Version: ap.Version,
			Type:    int(ap.Type),
			Key:     ap.Key,
			Index:   ap.Index,
		}
		defer ap.eventManager.Emit(pde)
	}
	hcfe := &control_event.HealthCheckFailedEvent{
		Name:    ap.Name,
		Version: ap.Version,
		Type:    int(ap.Type),
	}
	defer ap.eventManager.Emit(hcfe)
}

func (a *availablePlugin) HitCount() int {
	return a.hitCount
}

func (a *availablePlugin) LastHit() time.Time {
	return a.lastHitTime
}

// makeKey creates the ap.Key from the ap.Name and ap.Version
func (ap *availablePlugin) makeKey() {
	s := []string{ap.Name, strconv.Itoa(ap.Version)}
	ap.Key = strings.Join(s, ":")
}

// apCollection is a collection of availablePlugin
type apCollection struct {
	table       *map[string]*availablePluginPool
	mutex       *sync.Mutex
	keys        *[]string
	currentIter int
}

// newAPCollection returns an apCollection capable of storing availblePlugin
func newAPCollection() *apCollection {
	m := make(map[string]*availablePluginPool)
	var k []string
	return &apCollection{
		table:       &m,
		mutex:       &sync.Mutex{},
		keys:        &k,
		currentIter: 0,
	}
}

func (c *apCollection) GetPluginPool(key string) *availablePluginPool {
	c.Lock()
	defer c.Unlock()

	if ap, ok := (*c.table)[key]; ok {
		return ap
	}
	return nil
}

func (c *apCollection) PluginPoolHasAP(key string) bool {
	a := c.GetPluginPool(key)
	if a != nil && a.Count() > 0 {
		return true
	}
	return false
}

// Table returns a copy of the apCollection table
func (c *apCollection) Table() map[string][]*availablePlugin {
	c.Lock()
	defer c.Unlock()

	m := make(map[string][]*availablePlugin)
	for k, v := range *c.table {
		m[k] = *v.Plugins
	}
	return m
}

// Add adds an availablePlugin to the apCollection table
func (c *apCollection) Add(ap *availablePlugin) error {
	logger.Debugf("apcollection", "available plugin added %s", ap.String())
	c.Lock()
	defer c.Unlock()

	if _, ok := (*c.table)[ap.Key]; !ok {
		*c.keys = append(*c.keys, ap.Key)
	}

	if (*c.table)[ap.Key] != nil {
		// make sure we don't already have a pointer to this plugin in the table
		if exist, i := c.Exists(ap); exist {
			return errors.New("plugin instance already available at index " + strconv.Itoa(i))
		}
	} else {
		(*c.table)[ap.Key] = newAvailablePluginPool()
	}

	(*c.table)[ap.Key].Add(ap)
	return nil
}

// Remove removes an availablePlugin from the apCollection table
func (c *apCollection) Remove(ap *availablePlugin) error {
	c.Lock()
	defer c.Unlock()

	if exists, _ := c.Exists(ap); !exists {
		return errors.New("Warning: plugin does not exist in table")
	}

	(*c.table)[ap.Key].Remove(ap)
	logger.Debug("ap.removed", fmt.Sprintf(ap.Name, ap.Version))
	return nil
}

// Lock locks the mutex and is exported for external operations that may be unsafe
func (c *apCollection) Lock() {
	c.mutex.Lock()
}

// Unlock unlocks the mutex
func (c *apCollection) Unlock() {
	c.mutex.Unlock()
}

// Item returns the item at current position in the apCollection table
func (c *apCollection) Item() (string, *availablePluginPool) {
	key := (*c.keys)[c.currentIter-1]
	return key, (*c.table)[key]
}

// Next moves iteration position in the apCollection table
func (c *apCollection) Next() bool {
	c.currentIter++
	if c.currentIter > len(*c.table) {
		c.currentIter = 0
		return false
	}
	return true
}

// exists checks the table to see if a pointer for the availablePlugin specified
// already exists
func (c *apCollection) Exists(ap *availablePlugin) (bool, int) {
	return (*c.table)[ap.Key].Exists(ap)
}

// availablePlugins is a collection of availablePlugins by type
type availablePlugins struct {
	Collectors, Publishers, Processors *apCollection
}

// newAvailablePlugins returns an availablePlugins pointer
func newAvailablePlugins() *availablePlugins {
	return &availablePlugins{
		Collectors: newAPCollection(),
		Processors: newAPCollection(),
		Publishers: newAPCollection(),
	}
}

// Insert adds an availablePlugin into the correct collection based on type
func (a *availablePlugins) Insert(ap *availablePlugin) error {
	switch ap.Type {
	case plugin.CollectorPluginType:
		err := a.Collectors.Add(ap)
		return err
	case plugin.PublisherPluginType:
		err := a.Publishers.Add(ap)
		return err
	case plugin.ProcessorPluginType:
		err := a.Processors.Add(ap)
		return err
	default:
		return errors.New("cannot insert into available plugins, unknown plugin type")
	}
}

// Remove removes an availablePlugin from the correct collection based on type
func (a *availablePlugins) Remove(ap *availablePlugin) error {
	switch ap.Type {
	case plugin.CollectorPluginType:
		err := a.Collectors.Remove(ap)
		return err
	case plugin.PublisherPluginType:
		err := a.Publishers.Remove(ap)
		return err
	case plugin.ProcessorPluginType:
		err := a.Processors.Remove(ap)
		return err
	default:
		return errors.New("cannot remove from available plugins, unknown plugin type")
	}
}

type availablePluginPool struct {
	Plugins *[]*availablePlugin

	mutex *sync.Mutex
}

func newAvailablePluginPool() *availablePluginPool {
	app := make([]*availablePlugin, 0)
	return &availablePluginPool{
		Plugins: &app,
		mutex:   &sync.Mutex{},
	}
}

func (a *availablePluginPool) Lock() {
	a.mutex.Lock()
}

func (a *availablePluginPool) Unlock() {
	a.mutex.Unlock()
}

func (a *availablePluginPool) Count() int {
	return len((*a.Plugins))
}

func (a *availablePluginPool) Add(ap *availablePlugin) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	// tell ap its index in the table
	ap.Index = len((*a.Plugins))
	// append
	newCollection := append((*a.Plugins), ap)
	// overwrite
	a.Plugins = &newCollection
}

func (a *availablePluginPool) Remove(ap *availablePlugin) {
	a.Lock()
	defer a.Unlock()
	// Place nil here to allow GC per : https://github.com/golang/go/wiki/SliceTricks
	(*a.Plugins)[ap.Index] = nil
	splicedColl := append((*a.Plugins)[:ap.Index], (*a.Plugins)[ap.Index+1:]...)
	a.Plugins = &splicedColl
	//reset indexes
	a.resetIndexes()
}

func (a *availablePluginPool) Exists(ap *availablePlugin) (bool, int) {
	for i, _ap := range *a.Plugins {
		if ap == _ap {
			return true, i
		}
	}
	return false, -1
}

func (a *availablePluginPool) resetIndexes() {
	for i, ap := range *a.Plugins {
		ap.Index = i
	}
}

func (a *availablePluginPool) SelectUsingStrategy(strat RoutingStrategy) (*availablePlugin, error) {
	a.Lock()
	defer a.Unlock()

	sp := make([]routing.SelectablePlugin, len(*a.Plugins))
	for i, _ := range *a.Plugins {
		sp[i] = (*a.Plugins)[i]
	}
	sap, err := strat.Select(a, sp)
	if err != nil || sap == nil {
		return nil, err
	}
	return sap.(*availablePlugin), err
}
