# cronDemo
learn cron

#### 接口与匿名函数
定义一个接口类型`Job`，需要实现`Run()`方法。
```go
// Job is an interface for submitted cron jobs.
type Job interface {
	Run()
}
```
在方法中需要接口类型作为参数`func (c *Cron) AddFunc(spec string, cmd func())`，这时候需要创建实现类来传入。
```go
type testA struct{
}

func Run() {
}
```
可以创建一个实现类来将匿名函数作为参数
```go
// FuncJob is a wrapper that turns a func() into a cron.Job
type FuncJob func()

func (f FuncJob) Run() { f() }
```
这样在调用过程就可以直接使用匿名函数传入
```go
func (c *Cron) AddFunc(spec string, cmd func()) (EntryID, error) {
	return c.AddJob(spec, FuncJob(cmd))
}
```
#### 捕获panic
cron也可以捕获panic
```go

// Recover panics in wrapped jobs and log them with the provided logger.
func Recover(logger Logger) JobWrapper {
	return func(j Job) Job {
		return FuncJob(func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					logger.Error(err, "panic", "stack", "...\n"+string(buf))
				}
			}()
			j.Run()
		})
	}
}
```
#### next接口
`Next()`以入参的时间点作为标准，返回下次调度器激活的时间点。
```go
// Schedule describes a job's duty cycle.
type Schedule interface {
	// Next returns the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}
```
`ConstantDelaySchedule`对时间加一个固定的间隔，即固定周期使用调度器。
```go
// ConstantDelaySchedule represents a simple recurring duty cycle, e.g. "Every 5 minutes".
// It does not support jobs more frequent than once a second.
type ConstantDelaySchedule struct {
	Delay time.Duration
}
// Next returns the next time this should be run.
// This rounds so that the next activation time will be on the second.
func (schedule ConstantDelaySchedule) Next(t time.Time) time.Time {
	return t.Add(schedule.Delay - time.Duration(t.Nanosecond())*time.Nanosecond)
}
```
使用方式如下，可以设定固定时间段。
```go
func TestRunningMultipleSchedules(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	c := cron.New()
	c.AddFunc("0 0 0 1 1 ?", func() {})
	c.AddFunc("0 0 0 31 12 ?", func() {})
	c.AddFunc("* * * * * ?", func() { wg.Done() })
	c.Schedule(cron.Every(time.Minute), cron.FuncJob(func() {}))
	c.Schedule(cron.Every(time.Second), cron.FuncJob(func() { wg.Done() }))
	c.Schedule(cron.Every(time.Hour), cron.FuncJob(func() {}))

	c.Start()
	defer c.Stop()

	select {
	case <-time.After(2 * time.Second):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
```
`SpecSchedule`则是根据规定好的时间进行调度。
```go
func (s *SpecSchedule) Next(t time.Time) time.Time {}
```
