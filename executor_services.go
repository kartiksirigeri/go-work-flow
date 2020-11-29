package workflow

const (
	TASK_QUEUE_MAX   = 100
	CONCURRENT_TASKS = 50
)

type task func()

type ExecutorService struct {
	tasksQueue         chan task
	maxQueueSize       int
	maxConcurrentTasks int
}

var (
	default_es *ExecutorService
)

func init() {
	default_es = NewExecutorService(TASK_QUEUE_MAX, CONCURRENT_TASKS)
}

func (t task) execute() {
	t()
}

func NewExecutorService(queueSize, parallelTasks int) *ExecutorService {
	newService := &ExecutorService{
		tasksQueue:         make(chan task, queueSize),
		maxQueueSize:       queueSize,
		maxConcurrentTasks: parallelTasks,
	}
	newService.runJobs()
	return newService
}

func (e *ExecutorService) Submit(task task) {
	e.tasksQueue <- task
}

func (e *ExecutorService) runJobs() {
	for i := 0; i < e.maxConcurrentTasks; i++ {
		go func() {
			for t := range e.tasksQueue {
				t.execute()
			}
		}()
	}
}
