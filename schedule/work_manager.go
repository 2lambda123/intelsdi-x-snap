package schedule

import "sync"

/*
                             job queue
    workManager.Work(j) ----> [jjjjjjj]
                   ^                 |
                   |                 | workManager.sendToWorker(j)
                   |                 V
         job.Run() |            +---------+
 <-job.ReplyChan() |            | w  w  w |
                   +----------  |  w w  w | worker pool
                                |  w   w  |
                                +---------+

*/

type workManager struct {
	state          workManagerState
	collectq       *queue
	collectqSize   int64
	collectWkrs    []*worker
	collectWkrSize int
	collectchan    chan job
	kill           chan struct{}
	mutex          *sync.Mutex
}

type workManagerState int

const (
	workManagerStopped workManagerState = iota
	workManagerRunning
)

func newWorkManager(cqs int64, cws int) *workManager {

	wm := &workManager{
		collectWkrSize: cws,
		collectchan:    make(chan job),
		kill:           make(chan struct{}),
		mutex:          &sync.Mutex{},
	}

	wm.collectq = newQueue(cqs, wm.sendToWorker)
	wm.collectq.Start()

	wm.collectWkrs = make([]*worker, cws)
	for i := 0; i < cws; i++ {
		wm.collectWkrs[i] = newWorker(wm.collectchan)
		go wm.collectWkrs[i].start()
	}

	return wm
}

// workManager's loop just handles queuing errors.
func (w *workManager) Start() {

	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.state == workManagerStopped {
		w.state = workManagerRunning
		go func() {
			for {
				select {
				case qe := <-w.collectq.Err:
					qe.Job.ReplChan() <- struct{}{}
					//TODO: log error
				case <-w.kill:
					return
				}
			}
		}()
	}
}

func (w *workManager) Stop() {
	w.collectq.Stop()
	close(workerKillChan)
	close(w.kill)
}

// Work dispatches jobs to worker pools for processing.
// a job is queued, a worker receives it, and then replies
// on the job's  reply channel.
func (w *workManager) Work(j job) job {
	switch j.Type() {
	case collectJobType:
		w.collectq.Event <- j
	}
	<-j.ReplChan()
	return j
}

func (w *workManager) AddCollectWorker() {
	nw := newWorker(w.collectchan)
	go nw.start()
	w.collectWkrs = append(w.collectWkrs, nw)
	w.collectWkrSize++
}

// sendToWorker is the handler given to the queue.
// it dispatches work to the worker pool.
func (w *workManager) sendToWorker(j job) {
	switch j.Type() {
	case collectJobType:
		w.collectchan <- j
	}
}
