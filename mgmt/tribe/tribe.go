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

package tribe

import (
	"errors"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/pulse/core/perror"
	"github.com/pborman/uuid"

	"github.com/hashicorp/memberlist"
)

var (
	errAgreementDoesNotExist          = errors.New("Agreement does not exist")
	errAgreementAlreadyExists         = errors.New("Agreement already exists")
	errUnknownMember                  = errors.New("Unknown member")
	errAlreadyMemberOfPluginAgreement = errors.New("Already a member of a plugin agreement")
	errNotAMember                     = errors.New("Not a member of agreement")
	errTaskAlreadyExists              = errors.New("Task already exists")
	errTaskDoesNotExist               = errors.New("Task does not exist")
	errCreateMemberlist               = errors.New("Failed to create memberlist")
	errMemberlistJoin                 = errors.New("Failed to join memberlist")
)

var logger = log.WithFields(log.Fields{
	"_module": "tribe",
})

type agreements struct {
	PluginAgreement *pluginAgreement
	TaskAgreement   *taskAgreement
	Members         map[string]*member
}

type plugins []plugin

type pluginAgreement struct {
	Plugins plugins
}

type tasks []task

type taskAgreement struct {
	Tasks tasks
}

type task struct {
	ID string
}

type plugin struct {
	Name    string
	Version int
}

type tribe struct {
	clock        LClock
	agreements   map[string]*agreements
	mutex        sync.RWMutex
	msgBuffer    []msg
	intentBuffer []msg
	broadcasts   *memberlist.TransmitLimitedQueue
	memberlist   *memberlist.Memberlist
	logger       *log.Entry
	members      map[string]*member
}

type member struct {
	Name            string
	Node            *memberlist.Node
	PluginAgreement *pluginAgreement
	TaskAgreements  map[string]*taskAgreement
}

func newMember(node *memberlist.Node) *member {
	return &member{
		Name:           node.Name,
		Node:           node,
		TaskAgreements: map[string]*taskAgreement{},
	}
}

type config struct {
	seed             string
	memberlistConfig *memberlist.Config
}

func DefaultConfig(name, advertiseAddr string, advertisePort int, seed string) *config {
	c := &config{seed: seed}
	c.memberlistConfig = memberlist.DefaultLANConfig()
	c.memberlistConfig.PushPullInterval = 300 * time.Second
	c.memberlistConfig.Name = name
	c.memberlistConfig.BindAddr = advertiseAddr
	c.memberlistConfig.BindPort = advertisePort
	c.memberlistConfig.GossipNodes = c.memberlistConfig.GossipNodes * 2
	return c
}

func New(c *config) (*tribe, error) {
	tribe := &tribe{
		agreements:   map[string]*agreements{},
		members:      map[string]*member{},
		msgBuffer:    make([]msg, 512),
		intentBuffer: []msg{},
		logger:       logger.WithField("_name", c.memberlistConfig.Name),
	}

	tribe.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return len(tribe.memberlist.Members())
		},
		RetransmitMult: memberlist.DefaultLANConfig().RetransmitMult,
	}

	//configure delegates
	c.memberlistConfig.Delegate = &delegate{tribe: tribe}
	c.memberlistConfig.Events = &memberDelegate{tribe: tribe}

	ml, err := memberlist.Create(c.memberlistConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	tribe.memberlist = ml

	if c.seed != "" {
		_, err := ml.Join([]string{c.seed})
		if err != nil {
			log.Error(errMemberlistJoin)
			return nil, errMemberlistJoin
		}

	}

	return tribe, nil
}

func newAgreements() *agreements {
	return &agreements{
		PluginAgreement: &pluginAgreement{
			Plugins: plugins{},
		},
		TaskAgreement: &taskAgreement{
			Tasks: tasks{},
		},
		Members: map[string]*member{},
	}
}

// broadcast takes a tribe message type, encodes it for the wire, and queues
// the broadcast. If a notify channel is given, this channel will be closed
// when the broadcast is sent.
func (t *tribe) broadcast(mt msgType, msg interface{}, notify chan<- struct{}) error {
	raw, err := encodeMessage(mt, msg)
	if err != nil {
		return err
	}

	t.broadcasts.QueueBroadcast(&broadcast{
		msg:    raw,
		notify: notify,
	})
	return nil
}

func (t *tribe) LeaveAgreement(agreementName, memberName string) perror.PulseError {
	if err := t.canLeaveAgreement(agreementName, memberName); err != nil {
		return err
	}

	msg := &agreementMsg{
		LTime:         t.clock.Increment(),
		UUID:          uuid.New(),
		AgreementName: agreementName,
		MemberName:    memberName,
		Type:          leaveAgreementMsgType,
	}
	if t.handleLeaveAgreement(msg) {
		t.broadcast(leaveAgreementMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) JoinAgreement(agreementName, memberName string) perror.PulseError {
	if err := t.canJoinAgreement(agreementName, memberName); err != nil {
		return err
	}

	msg := &agreementMsg{
		LTime:         t.clock.Increment(),
		UUID:          uuid.New(),
		AgreementName: agreementName,
		MemberName:    memberName,
		Type:          joinAgreementMsgType,
	}
	if t.handleJoinAgreement(msg) {
		t.broadcast(joinAgreementMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) AddPlugin(agreementName, pluginName string, ver int) error {
	if _, ok := t.agreements[agreementName]; !ok {
		return errAgreementDoesNotExist
	}
	msg := &pluginMsg{
		LTime:         t.clock.Increment(),
		Plugin:        plugin{Name: pluginName, Version: ver},
		AgreementName: agreementName,
		UUID:          uuid.New(),
		Type:          addPluginMsgType,
	}
	if t.handleAddPlugin(msg) {
		t.broadcast(addPluginMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) RemovePlugin(clan, name string, ver int) error {
	if _, ok := t.agreements[clan]; !ok {
		return errAgreementDoesNotExist
	}
	msg := &pluginMsg{
		LTime:         t.clock.Increment(),
		Plugin:        plugin{Name: name, Version: ver},
		AgreementName: clan,
		UUID:          uuid.New(),
		Type:          removePluginMsgType,
	}
	if t.handleRemovePlugin(msg) {
		t.broadcast(removePluginMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) AddTask(agreementName, taskID string) perror.PulseError {
	if err := t.canAddTask(task{ID: taskID}, agreementName); err != nil {
		return err
	}
	msg := &taskMsg{
		LTime:         t.clock.Increment(),
		TaskID:        taskID,
		AgreementName: agreementName,
		UUID:          uuid.New(),
		Type:          addTaskMsgType,
	}
	if t.handleAddTask(msg) {
		t.broadcast(addTaskMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) RemoveTask(agreementName, taskID string) perror.PulseError {
	if err := t.canRemoveTask(task{ID: taskID}, agreementName); err != nil {
		return err
	}
	msg := &taskMsg{
		LTime:         t.clock.Increment(),
		TaskID:        taskID,
		AgreementName: agreementName,
		UUID:          uuid.New(),
		Type:          removeTaskMsgType,
	}
	if t.handleRemoveTask(msg) {
		t.broadcast(removeTaskMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) AddAgreement(name string) perror.PulseError {
	if _, ok := t.agreements[name]; ok {
		fields := log.Fields{
			"agreement": name,
		}
		return perror.New(errAgreementAlreadyExists, fields)
	}
	msg := &agreementMsg{
		LTime:         t.clock.Increment(),
		AgreementName: name,
		UUID:          uuid.New(),
		Type:          addAgreementMsgType,
	}
	if t.handleAddAgreement(msg) {
		t.broadcast(addAgreementMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) RemoveAgreement(name string) perror.PulseError {
	if _, ok := t.agreements[name]; !ok {
		fields := log.Fields{
			"Agreement": name,
		}
		return perror.New(errAgreementDoesNotExist, fields)
	}
	msg := &agreementMsg{
		LTime:         t.clock.Increment(),
		AgreementName: name,
		UUID:          uuid.New(),
		Type:          removeAgreementMsgType,
	}
	if t.handleRemoveAgreement(msg) {
		t.broadcast(removeAgreementMsgType, msg, nil)
	}
	return nil
}

func (t *tribe) processIntents() {
	for {
		if t.processAddPluginIntents() &&
			t.processRemovePluginIntents() &&
			t.processAddAgreementIntents() &&
			t.processRemoveAgreementIntents() &&
			t.processJoinAgreementIntents() &&
			t.processLeaveAgreementIntents() &&
			t.processAddTaskIntents() &&
			t.processRemoveTaskIntents() {
			return
		}
	}
}

func (t *tribe) processAddPluginIntents() bool {
	for idx, v := range t.intentBuffer {
		if v.GetType() == addPluginMsgType {
			intent := v.(*pluginMsg)
			if _, ok := t.agreements[intent.AgreementName]; ok {
				if ok, _ := t.agreements[intent.AgreementName].PluginAgreement.Plugins.contains(intent.Plugin); !ok {
					t.agreements[intent.AgreementName].PluginAgreement.Plugins = append(t.agreements[intent.AgreementName].PluginAgreement.Plugins, intent.Plugin)
					t.intentBuffer = append(t.intentBuffer[:idx], t.intentBuffer[idx+1:]...)
					return false
				}
			}
		}
	}
	return true
}

func (t *tribe) processRemovePluginIntents() bool {
	for k, v := range t.intentBuffer {
		if v.GetType() == removePluginMsgType {
			intent := v.(*pluginMsg)
			if a, ok := t.agreements[intent.AgreementName]; ok {
				if ok, idx := a.PluginAgreement.Plugins.contains(intent.Plugin); ok {
					a.PluginAgreement.Plugins = append(a.PluginAgreement.Plugins[:idx], a.PluginAgreement.Plugins[idx+1:]...)
					t.intentBuffer = append(t.intentBuffer[:k], t.intentBuffer[k+1:]...)
					return false
				}
			}
		}
	}
	return true
}

func (t *tribe) processAddTaskIntents() bool {
	for idx, v := range t.intentBuffer {
		if v.GetType() == addTaskMsgType {
			intent := v.(*taskMsg)
			if a, ok := t.agreements[intent.AgreementName]; ok {
				if ok, _ := a.TaskAgreement.Tasks.contains(task{ID: intent.TaskID}); !ok {
					a.TaskAgreement.Tasks = append(a.TaskAgreement.Tasks, task{ID: intent.TaskID})
					t.intentBuffer = append(t.intentBuffer[:idx], t.intentBuffer[idx+1:]...)
					return false
				}
			}
		}
	}
	return true
}

func (t *tribe) processRemoveTaskIntents() bool {
	for k, v := range t.intentBuffer {
		if v.GetType() == removeTaskMsgType {
			intent := v.(*taskMsg)
			if _, ok := t.agreements[intent.AgreementName]; ok {
				if ok, idx := t.agreements[intent.AgreementName].TaskAgreement.Tasks.contains(task{ID: intent.TaskID}); ok {
					t.agreements[intent.AgreementName].TaskAgreement.Tasks = append(t.agreements[intent.AgreementName].TaskAgreement.Tasks[:idx], t.agreements[intent.AgreementName].TaskAgreement.Tasks[idx+1:]...)
					t.intentBuffer = append(t.intentBuffer[:k], t.intentBuffer[k+1:]...)
					return false
				}
			}
		}
	}
	return true
}

func (t *tribe) processAddAgreementIntents() bool {
	for idx, v := range t.intentBuffer {
		if v.GetType() == addAgreementMsgType {
			intent := v.(*agreementMsg)
			if _, ok := t.agreements[intent.AgreementName]; !ok {
				t.agreements[intent.AgreementName] = newAgreements()
				t.intentBuffer = append(t.intentBuffer[:idx], t.intentBuffer[idx+1:]...)
				return false
			}
		}
	}
	return true
}

func (t *tribe) processRemoveAgreementIntents() bool {
	for k, v := range t.intentBuffer {
		if v.GetType() == removeAgreementMsgType {
			intent := v.(*agreementMsg)
			if _, ok := t.agreements[intent.Agreement()]; ok {
				delete(t.agreements, intent.Agreement())
				t.intentBuffer = append(t.intentBuffer[:k], t.intentBuffer[k+1:]...)
				return false
			}
		}
	}
	return true
}

func (t *tribe) processJoinAgreementIntents() bool {
	for idx, v := range t.intentBuffer {
		if v.GetType() == joinAgreementMsgType {
			intent := v.(*agreementMsg)
			if _, ok := t.members[intent.MemberName]; ok {
				if _, ok := t.agreements[intent.AgreementName]; ok {
					err := t.joinAgreement(intent)
					if err == nil {
						t.intentBuffer = append(t.intentBuffer[:idx], t.intentBuffer[idx+1:]...)
					}
					return false
				}
			}
		}
	}
	return true
}

func (t *tribe) processLeaveAgreementIntents() bool {
	for idx, v := range t.intentBuffer {
		if v.GetType() == joinAgreementMsgType {
			intent := v.(*agreementMsg)
			if _, ok := t.members[intent.MemberName]; ok {
				if _, ok := t.agreements[intent.AgreementName]; ok {
					if _, ok := t.agreements[intent.AgreementName].Members[intent.MemberName]; ok {
						err := t.leaveAgreement(intent)
						if err == nil {
							t.intentBuffer = append(t.intentBuffer[:idx], t.intentBuffer[idx+1:]...)
						}
						return false
					}
				}
			}
		}
	}
	return true
}

func (t *tribe) handleRemovePlugin(msg *pluginMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if _, ok := t.agreements[msg.Agreement()]; ok {
		if t.agreements[msg.AgreementName].PluginAgreement.remove(msg, t.logger) {
			t.processIntents()
			return true
		}
	}

	t.addPluginIntent(msg)
	return true
}

func (t *tribe) handleAddPlugin(msg *pluginMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if _, ok := t.agreements[msg.AgreementName]; ok {
		if t.agreements[msg.AgreementName].PluginAgreement.add(msg, t.logger) {
			t.processIntents()
			return true
		}
	}

	t.addPluginIntent(msg)
	return true
}

func (t *tribe) handleAddTask(msg *taskMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if _, ok := t.agreements[msg.AgreementName]; ok {
		if t.agreements[msg.AgreementName].TaskAgreement.add(msg, t.logger) {
			t.processIntents()
			return true
		}
	}

	t.addTaskIntent(msg)
	return true
}

func (t *tribe) handleRemoveTask(msg *taskMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if _, ok := t.agreements[msg.Agreement()]; ok {
		if t.agreements[msg.AgreementName].TaskAgreement.remove(msg, t.logger) {
			t.processIntents()
			return true
		}
	}

	t.addTaskIntent(msg)
	return true
}

func (t *tribe) handleMemberJoin(n *memberlist.Node) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.members[n.Name]; !ok {
		t.members[n.Name] = newMember(n)
	}
	t.processIntents()
}

func (t *tribe) handleMemberLeave(n *memberlist.Node) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.members[n.Name]; ok {
		delete(t.members, n.Name)
	}
}

func (t *tribe) handleMemberUpdate(n *memberlist.Node) {

}

func (t *tribe) handleAddAgreement(msg *agreementMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	// add msg to seen buffer
	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	// add agreement
	if _, ok := t.agreements[msg.AgreementName]; !ok {
		t.agreements[msg.AgreementName] = newAgreements()
		t.processIntents()
		return true
	}
	t.addAgreementIntent(msg)
	return true
}

func (t *tribe) handleRemoveAgreement(msg *agreementMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	// add msg to seen buffer
	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if _, ok := t.agreements[msg.AgreementName]; ok {
		delete(t.agreements, msg.AgreementName)
		t.processIntents()
		// TODO consider removing any intents that involve this agreement
		return true
	}

	return true
}

func (t *tribe) handleJoinAgreement(msg *agreementMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if err := t.joinAgreement(msg); err == nil {
		t.processIntents()
		return true
	}

	t.addAgreementIntent(msg)
	return true
}

func (t *tribe) handleLeaveAgreement(msg *agreementMsg) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// update the clock if newer
	t.clock.Update(msg.LTime)

	if t.isDuplicate(msg) {
		return false
	}

	t.msgBuffer[msg.LTime%LTime(len(t.msgBuffer))] = msg

	if err := t.leaveAgreement(msg); err == nil {
		t.processIntents()
		return true
	}

	t.addAgreementIntent(msg)

	return true
}

func (t *tribe) joinAgreement(msg *agreementMsg) perror.PulseError {
	if err := t.canJoinAgreement(msg.Agreement(), msg.MemberName); err != nil {
		return err
	}
	// add plugin agreement to the member
	if t.agreements[msg.Agreement()].PluginAgreement != nil {
		t.members[msg.MemberName].PluginAgreement = t.agreements[msg.Agreement()].PluginAgreement
	}
	t.members[msg.MemberName].TaskAgreements[msg.Agreement()] = t.agreements[msg.Agreement()].TaskAgreement

	// update the agreements membership
	t.agreements[msg.Agreement()].Members[msg.MemberName] = t.members[msg.MemberName]
	return nil
}

func (t *tribe) leaveAgreement(msg *agreementMsg) perror.PulseError {
	if err := t.canLeaveAgreement(msg.Agreement(), msg.MemberName); err != nil {
		return err
	}

	delete(t.agreements[msg.AgreementName].Members, msg.MemberName)
	t.members[msg.MemberName].PluginAgreement = nil
	if _, ok := t.members[msg.MemberName].TaskAgreements[msg.Agreement()]; ok {
		delete(t.members[msg.MemberName].TaskAgreements, msg.Agreement())
	}

	return nil
}

func (t *tribe) canLeaveAgreement(agreementName, memberName string) perror.PulseError {
	if _, ok := t.agreements[agreementName]; !ok {
		fields := log.Fields{
			"Agreement": agreementName,
		}
		logger.WithFields(fields).Debugln(errAgreementDoesNotExist)
		return perror.New(errAgreementDoesNotExist, fields)
	}

	m, ok := t.members[memberName]
	if !ok {
		fields := log.Fields{
			"MemberName": memberName,
		}
		t.logger.WithFields(fields).Debugln(errUnknownMember)
		return perror.New(errUnknownMember, fields)
	}
	if m.PluginAgreement == nil {
		fields := log.Fields{
			"MemberName": t.memberlist.LocalNode().Name,
			"Agreement":  agreementName,
		}
		t.logger.WithFields(fields).Debugln(errNotAMember)
		return perror.New(errNotAMember, fields)
	}
	return nil
}

func (t *tribe) canJoinAgreement(agreementName, memberName string) perror.PulseError {
	if _, ok := t.agreements[agreementName]; !ok {
		fields := log.Fields{
			"Agreement": agreementName,
		}
		logger.WithFields(fields).Debugln(errAgreementDoesNotExist)
		return perror.New(errAgreementDoesNotExist, fields)
	}
	m, ok := t.members[memberName]
	if !ok {
		fields := log.Fields{
			"MemberName": memberName,
		}
		t.logger.WithFields(fields).Debugln(errUnknownMember)
		return perror.New(errUnknownMember, fields)

	}
	if m.PluginAgreement != nil && len(m.PluginAgreement.Plugins) > 0 {
		fields := log.Fields{
			"MemberName": t.memberlist.LocalNode().Name,
			"Agreement":  agreementName,
		}
		t.logger.WithFields(fields).Debugln(errAlreadyMemberOfPluginAgreement)
		return perror.New(errAlreadyMemberOfPluginAgreement, fields)
	}
	return nil
}

func (t *tribe) canAddTask(task task, agreementName string) perror.PulseError {
	fields := log.Fields{
		"Agreement": agreementName,
	}
	a, ok := t.agreements[agreementName]
	if !ok {
		logger.WithFields(fields).Debugln(errAgreementDoesNotExist)
		return perror.New(errAgreementDoesNotExist, fields)
	}
	if ok, _ := a.TaskAgreement.Tasks.contains(task); ok {
		logger.WithFields(fields).Debugln(errTaskAlreadyExists)
		return perror.New(errTaskAlreadyExists, fields)
	}
	return nil
}

func (t *tribe) canRemoveTask(task task, agreementName string) perror.PulseError {
	fields := log.Fields{
		"Agreement": agreementName,
	}
	a, ok := t.agreements[agreementName]
	if !ok {
		logger.WithFields(fields).Debugln(errAgreementDoesNotExist)
		return perror.New(errAgreementDoesNotExist, fields)
	}
	if ok, _ := a.TaskAgreement.Tasks.contains(task); !ok {
		logger.WithFields(fields).Debugln(errTaskDoesNotExist)
		return perror.New(errTaskDoesNotExist, fields)
	}
	return nil
}

func (t *tribe) isDuplicate(msg msg) bool {
	// is the message old
	if t.clock.Time() > LTime(len(t.msgBuffer)) &&
		msg.Time() < t.clock.Time()-LTime(len(t.msgBuffer)) {
		t.logger.WithFields(log.Fields{
			"event_clock": msg.Time(),
			"event":       msg.GetType().String(),
			"event_uuid":  msg.ID(),
			"clock":       t.clock.Time(),
			"agreement":   msg.Agreement(),
			// "plugin":      msg.Plugin,
		}).Debugln("This message is old")
		return true
	}

	// have we seen it
	idx := msg.Time() % LTime(len(t.msgBuffer))
	seen := t.msgBuffer[idx]
	if seen != nil && seen.ID() == msg.ID() {
		t.logger.WithFields(log.Fields{
			"event_clock": msg.Time(),
			"event":       msg.GetType().String(),
			"event_uuid":  msg.ID(),
			"clock":       t.clock.Time(),
			"agreement":   msg.Agreement(),
			// "plugin":      msg.Plugin,
		}).Debugln("duplicate message")

		return true
	}
	return false
}

// contains - Returns boolean indicating whether the plugin was found.
// If the plugin is found the index returned as the second return value.
func (p plugins) contains(item plugin) (bool, int) {
	for idx, i := range p {
		if i.Name == item.Name && i.Version == item.Version {
			return true, idx
		}
	}
	return false, -1
}

// contains - Returns boolean indicating whether the plugin was found.
// If the plugin is found the index returned as the second return value.
func (t tasks) contains(item task) (bool, int) {
	for idx, i := range t {
		if i.ID == item.ID {
			return true, idx
		}
	}
	return false, -1
}

func (a *pluginAgreement) remove(msg *pluginMsg, tlogger *log.Entry) bool {
	tlogger.WithFields(log.Fields{
		"event_clock": msg.LTime,
		"event":       msg.Type.String(),
		"agreement":   msg.AgreementName,
		"plugin":      msg.Plugin,
	}).Debugln("Removing plugin")
	if ok, idx := a.Plugins.contains(msg.Plugin); ok {
		a.Plugins = append(a.Plugins[idx+1:], a.Plugins[:idx]...)
		return true
	}
	return false
}

func (a *pluginAgreement) add(msg *pluginMsg, tlogger *log.Entry) bool {
	tlogger.WithFields(log.Fields{
		"event_clock": msg.LTime,
		"agreement":   msg.AgreementName,
		"plugin":      msg.Plugin,
		"_block":      "add",
	}).Debugln("Adding plugin")
	if ok, _ := a.Plugins.contains(msg.Plugin); ok {
		return false
	}
	a.Plugins = append(a.Plugins, msg.Plugin)
	return true
}

func (a *taskAgreement) add(msg *taskMsg, tlogger *log.Entry) bool {
	tlogger.WithFields(log.Fields{
		"event_clock": msg.LTime,
		"agreement":   msg.AgreementName,
		"task_id":     msg.TaskID,
	}).Debugln("Adding task")
	if ok, _ := a.Tasks.contains(task{ID: msg.TaskID}); ok {
		return false
	}
	a.Tasks = append(a.Tasks, task{ID: msg.TaskID})
	return true
}

func (a *taskAgreement) remove(msg *taskMsg, tlogger *log.Entry) bool {
	tlogger.WithFields(log.Fields{
		"event_clock": msg.LTime,
		"event":       msg.Type.String(),
		"agreement":   msg.AgreementName,
	}).Debugln("Removing task")
	if ok, idx := a.Tasks.contains(task{ID: msg.TaskID}); ok {
		a.Tasks = append(a.Tasks[idx+1:], a.Tasks[:idx]...)
		return true
	}
	return false
}

func (t *tribe) addPluginIntent(msg *pluginMsg) bool {
	t.logger.WithFields(log.Fields{
		"event_clock": msg.LTime,
		"agreement":   msg.AgreementName,
		"plugin":      msg.Plugin,
		"type":        msg.Type.String(),
	}).Debugln("Out of order msg")
	t.intentBuffer = append(t.intentBuffer, msg)
	return true
}

func (t *tribe) addAgreementIntent(m msg) bool {
	t.logger.WithFields(log.Fields{
		"event_clock": m.Time(),
		"agreement":   m.Agreement(),
		"type":        m.GetType().String(),
	}).Debugln("Out of order msg")
	t.intentBuffer = append(t.intentBuffer, m)
	return true
}

func (t *tribe) addTaskIntent(m *taskMsg) bool {
	t.logger.WithFields(log.Fields{
		"event_clock": m.Time(),
		"agreement":   m.Agreement(),
		"type":        m.GetType().String(),
		"task_id":     m.TaskID,
	}).Debugln("Out of order msg")
	t.intentBuffer = append(t.intentBuffer, m)
	return true
}