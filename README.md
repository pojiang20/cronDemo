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

### cron的run
下面的代码对run过程做了部分删减，主体逻辑如下：
1. 将当前时间分配给所有实体
2. 进入for循环，对实体以时间先后来排序
3. 选取第一个实体（最早的）以当前时间和调度时间的差来设置定时器
4. for+select进入小循环：
   a. 如果计时器到期触发，则执行任务再跳出循环
   b. 如果有新任务添加触发，则添加任务跳出循环
   c. 如果有查询请求触发，则返回查询结果继续等待触发
   d. 暂停触发，退出函数
   e. 移除任务请求触发，移除任务退出小循环
5. 重复执行大循环，即排序、定时和进入小循环等待被触发。
```go
func (c *Cron) run() {
	c.logger.Info("start")

	// Figure out the next activation times for each entry.
	now := c.now()
	for _, entry := range c.entries {
		entry.Next = entry.Schedule.Next(now)
	}

	for {
		// Determine the next entry to run.
		sort.Sort(byTime(c.entries))

		var timer *time.Timer
		timer = time.NewTimer(c.entries[0].Next.Sub(now))

		for {
			select {
			case now = <-timer.C:

			case newEntry := <-c.add:

			case replyChan := <-c.snapshot:

			case <-c.stop:

			case id := <-c.remove:
			}

			break
		}
	}
}
```

### 参考
https://eddycjy.gitbook.io/golang/di-3-ke-gin/cron
https://darjun.github.io/2020/06/25/godailylib/cron/