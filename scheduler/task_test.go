package scheduler

import (
	"testing"
	"time"

	"github.com/intelsdi-x/pulse/control/plugin"
	"github.com/intelsdi-x/pulse/core"
	"github.com/intelsdi-x/pulse/pkg/schedule"
	"github.com/intelsdi-x/pulse/scheduler/wmap"

	log "github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/gomit"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	emitter = gomit.NewEventController()
)

func TestTask(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	Convey("Task", t, func() {
		sampleWFMap := wmap.Sample()
		wf, errs := wmapToWorkflow(sampleWFMap)
		So(errs, ShouldBeEmpty)
		c := &mockMetricManager{}
		c.setAcceptedContentType("rabbitmq", core.PublisherPluginType, 5, []string{plugin.PulseGOBContentType})
		err := wf.BindPluginContentTypes(c)
		So(err, ShouldBeNil)
		Convey("task + simple schedule", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task.Spin()
			time.Sleep(time.Millisecond * 10) // it is a race so we slow down the test
			So(task.state, ShouldEqual, core.TaskSpinning)

			task.Stop()
		})

		Convey("Task specified-name test", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter, core.SetTaskName("My name is unique"))
			task.Spin()
			So(task.GetName(), ShouldResemble, "My name is unique")

		})
		Convey("Task default-name test", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task.Spin()
			So(task.GetName(), ShouldResemble, "Task-"+task.ID())

		})

		Convey("Task deadline duration test", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter, core.TaskDeadlineDuration(20*time.Second))
			task.Spin()
			So(task.deadlineDuration, ShouldEqual, 20*time.Second)
			task.Option(core.TaskDeadlineDuration(20 * time.Second))

			So(core.TaskDeadlineDuration(2*time.Second), ShouldNotBeEmpty)

		})

		Convey("Tasks are created and creation of task table is checked", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task1 := newTask(sch, wf, newWorkManager(), c, emitter)
			task1.Spin()
			task.Spin()
			tC := newTaskCollection()
			tC.add(task)
			tC.add(task1)
			taskTable := tC.Table()

			So(len(taskTable), ShouldEqual, 2)

		})

		Convey("Task is created and starts to spin", func() {
			sch := schedule.NewSimpleSchedule(time.Second * 5)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task.Spin()
			So(task.state, ShouldEqual, core.TaskSpinning)
			Convey("Task is Stopped", func() {
				task.Stop()
				time.Sleep(time.Millisecond * 10) // it is a race so we slow down the test
				So(task.state, ShouldEqual, core.TaskStopped)
			})
		})

		Convey("task fires", func() {
			sch := schedule.NewSimpleSchedule(time.Nanosecond * 100)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task.Spin()
			time.Sleep(time.Millisecond * 50)
			So(task.hitCount, ShouldBeGreaterThan, 2)
			So(task.missedIntervals, ShouldBeGreaterThan, 2)
			task.Stop()
		})

		Convey("Enable a running task", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 10)
			task := newTask(sch, wf, newWorkManager(), c, emitter)
			task.Spin()
			err := task.Enable()
			So(err, ShouldNotBeNil)
			So(task.State(), ShouldEqual, core.TaskSpinning)
		})

		Convey("Enable a diabled task", func() {
			sch := schedule.NewSimpleSchedule(time.Millisecond * 10)
			task := newTask(sch, wf, newWorkManager(), c, emitter)

			task.state := core.TaskDisabled
			err := task.Enable()
			So(err, ShouldBeNil)
			So(task.State(), ShouldEqual, core.TaskStopped)
		})
	})

	Convey("Create task collection", t, func() {
		sampleWFMap := wmap.Sample()
		wf, errs := wmapToWorkflow(sampleWFMap)
		So(errs, ShouldBeEmpty)

		sch := schedule.NewSimpleSchedule(time.Millisecond * 10)
		task := newTask(sch, wf, newWorkManager(), &mockMetricManager{}, emitter)
		So(task.id, ShouldNotBeEmpty)
		So(task.id, ShouldNotBeNil)
		taskCollection := newTaskCollection()

		Convey("Add task to collection", func() {

			err := taskCollection.add(task)
			So(err, ShouldBeNil)
			So(len(taskCollection.table), ShouldEqual, 1)

			Convey("Attempt to add the same task again", func() {
				err := taskCollection.add(task)
				So(err, ShouldNotBeNil)
			})

			Convey("Get task from collection", func() {
				t := taskCollection.Get(task.id)
				So(t, ShouldNotBeNil)
				So(t.ID(), ShouldEqual, task.id)
				So(t.CreationTime().Nanosecond(), ShouldBeLessThan, time.Now().Nanosecond())
				So(t.HitCount(), ShouldEqual, 0)
				So(t.MissedCount(), ShouldEqual, 0)
				So(t.State(), ShouldEqual, core.TaskStopped)
				So(t.Status(), ShouldEqual, core.WorkflowStopped)
				So(t.LastRunTime().IsZero(), ShouldBeTrue)
			})

			Convey("Attempt to get task with an invalid Id", func() {
				t := taskCollection.Get("1234")
				So(t, ShouldBeNil)
			})

			Convey("Create another task and compare the id", func() {
				task2 := newTask(sch, wf, newWorkManager(), &mockMetricManager{}, emitter)
				So(task2.id, ShouldNotEqual, task.ID())
			})

		})

	})
}
