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

package rbody

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/intelsdi-x/pulse/core"
	"github.com/intelsdi-x/pulse/mgmt/rest/request"
	"github.com/intelsdi-x/pulse/pkg/schedule"
	"github.com/intelsdi-x/pulse/scheduler/wmap"
)

const (
	ScheduledTaskListReturnedType  = "scheduled_task_list_returned"
	ScheduledTaskReturnedType      = "scheduled_task_returned"
	AddScheduledTaskType           = "scheduled_task_created"
	ScheduledTaskType              = "scheduled_task"
	ScheduledTaskStartedType       = "scheduled_task_started"
	ScheduledTaskStoppedType       = "scheduled_task_stopped"
	ScheduledTaskRemovedType       = "scheduled_task_removed"
	ScheduledTaskWatchingEndedType = "schedule_task_watch_ended"

	// Event types for task watcher streaming
	TaskWatchStreamOpen   = "stream-open"
	TaskWatchMetricEvent  = "metric-event"
	TaskWatchTaskDisabled = "task-disabled"
	TaskWatchTaskStarted  = "task-started"
	TaskWatchTaskStopped  = "task-stopped"
)

type ScheduledTaskListReturned struct {
	ScheduledTasks []ScheduledTask
}

func (s *ScheduledTaskListReturned) Len() int {
	return len(s.ScheduledTasks)
}

func (s *ScheduledTaskListReturned) Less(i, j int) bool {
	return s.ScheduledTasks[i].Name < s.ScheduledTasks[j].Name
}

func (s *ScheduledTaskListReturned) Swap(i, j int) {
	s.ScheduledTasks[i], s.ScheduledTasks[j] = s.ScheduledTasks[j], s.ScheduledTasks[i]
}

func (s *ScheduledTaskListReturned) ResponseBodyMessage() string {
	return "Scheduled tasks retrieved"
}

func (s *ScheduledTaskListReturned) ResponseBodyType() string {
	return ScheduledTaskListReturnedType
}

type ScheduledTaskReturned struct {
	AddScheduledTask
}

func (s *ScheduledTaskReturned) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task (%d) returned", s.ID)
}

func (s *ScheduledTaskReturned) ResponseBodyType() string {
	return ScheduledTaskReturnedType
}

type AddScheduledTask ScheduledTask

func (s *AddScheduledTask) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task created (%d)", s.ID)
}

func (s *AddScheduledTask) ResponseBodyType() string {
	return AddScheduledTaskType
}

func AddSchedulerTaskFromTask(t core.Task) *AddScheduledTask {
	st := &AddScheduledTask{
		ID:                 int(t.ID()),
		Name:               t.GetName(),
		Deadline:           t.DeadlineDuration().String(),
		CreationTimestamp:  t.CreationTime().Unix(),
		LastRunTimestamp:   t.LastRunTime().Unix(),
		HitCount:           int(t.HitCount()),
		MissCount:          int(t.MissedCount()),
		FailedCount:        int(t.FailedCount()),
		LastFailureMessage: t.LastFailureMessage(),
		State:              t.State().String(),
		Workflow:           t.WMap(),
	}
	assertSchedule(t.Schedule(), st)
	if st.LastRunTimestamp < 0 {
		st.LastRunTimestamp = -1
	}
	return st
}

type ScheduledTask struct {
	ID                 int               `json:"id"`
	Name               string            `json:"name"`
	Deadline           string            `json:"deadline"`
	Workflow           *wmap.WorkflowMap `json:"workflow,omitempty"`
	Schedule           *request.Schedule `json:"schedule,omitempty"`
	CreationTimestamp  int64             `json:"creation_timestamp,omitempty"`
	LastRunTimestamp   int64             `json:"last_run_timestamp,omitempty"`
	HitCount           int               `json:"hit_count,omitempty"`
	MissCount          int               `json:"miss_count,omitempty"`
	FailedCount        int               `json:"failed_count,omitempty"`
	LastFailureMessage string            `json:"last_failure_message,omitempty"`
	State              string            `json:"task_state"`
}

func (s *ScheduledTask) CreationTime() time.Time {
	return time.Unix(s.CreationTimestamp, 0)
}

func (s *ScheduledTask) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task created (%d)", s.ID)
}

func (s *ScheduledTask) ResponseBodyType() string {
	return ScheduledTaskType
}

func SchedulerTaskFromTask(t core.Task) *ScheduledTask {
	st := &ScheduledTask{
		ID:                 int(t.ID()),
		Name:               t.GetName(),
		Deadline:           t.DeadlineDuration().String(),
		CreationTimestamp:  t.CreationTime().Unix(),
		LastRunTimestamp:   t.LastRunTime().Unix(),
		HitCount:           int(t.HitCount()),
		MissCount:          int(t.MissedCount()),
		FailedCount:        int(t.FailedCount()),
		LastFailureMessage: t.LastFailureMessage(),
		State:              t.State().String(),
	}
	if st.LastRunTimestamp < 0 {
		st.LastRunTimestamp = -1
	}
	return st
}

type ScheduledTaskStarted struct {
	// TODO return resource
	ID int `json:"id"`
}

func (s *ScheduledTaskStarted) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task (%d) started", s.ID)
}

func (s *ScheduledTaskStarted) ResponseBodyType() string {
	return ScheduledTaskStartedType
}

type ScheduledTaskStopped struct {
	// TODO return resource
	ID int `json:"id"`
}

func (s *ScheduledTaskStopped) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task (%d) stopped", s.ID)
}

func (s *ScheduledTaskStopped) ResponseBodyType() string {
	return ScheduledTaskStoppedType
}

type ScheduledTaskRemoved struct {
	// TODO return resource
	ID int `json:"id"`
}

func (s *ScheduledTaskRemoved) ResponseBodyMessage() string {
	return fmt.Sprintf("Scheduled task (%d) removed", s.ID)
}

func (s *ScheduledTaskRemoved) ResponseBodyType() string {
	return ScheduledTaskRemovedType
}

func assertSchedule(s schedule.Schedule, t *AddScheduledTask) {
	switch v := s.(type) {
	case *schedule.SimpleSchedule:
		t.Schedule = &request.Schedule{
			Type:     "simple",
			Interval: v.Interval.String(),
		}
		return
	}
	t.Schedule = &request.Schedule{}
}

type ScheduledTaskWatchingEnded struct {
}

func (s *ScheduledTaskWatchingEnded) ResponseBodyMessage() string {
	return "Task watching ended"
}

func (s *ScheduledTaskWatchingEnded) ResponseBodyType() string {
	return ScheduledTaskWatchingEndedType
}

type StreamedTaskEvent struct {
	// Used to describe the event
	EventType string           `json:"type"`
	Message   string           `json:"message"`
	Event     []StreamedMetric `json:"event,omitempty"`
}

func (s *StreamedTaskEvent) ToJSON() string {
	j, _ := json.Marshal(s)
	return string(j)
}

type StreamedMetric struct {
	Namespace string      `json:"namespace"`
	Data      interface{} `json:"data"`
	Source    string      `json:"source"`
	Timestamp time.Time   `json:"timestamp"`
}
