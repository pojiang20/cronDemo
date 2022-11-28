package main

import (
	"fmt"
	"github.com/robfig/cron"
	"sync"
	"testing"
	"time"
)

func Test_AddFunc(t *testing.T) {
	c := cron.New()

	c.AddFunc("@every 1s", func() {
		fmt.Println("now: ", time.Now().String())
	})

	c.Start()
	time.Sleep(time.Second * 5)
}

type panicJob struct {
	count int
}

func (p *panicJob) Run() {
	p.count++
	//第一次调用panic
	if p.count == 1 {
		panic("oooooooooooooops!!!")
	}

	fmt.Println("hello world")
}

func Test_Recover(t *testing.T) {
	c := cron.New()
	c.AddJob("@every 1s", cron.NewChain(cron.Recover(cron.DefaultLogger)).Then(&panicJob{}))
	c.Start()

	time.Sleep(5 * time.Second)
}

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

// Test that the cron is run in the local time zone (as opposed to UTC).
func TestLocalTimezone(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	now := time.Now()
	// FIX: Issue #205
	// This calculation doesn't work in seconds 58 or 59.
	// Take the easy way out and sleep.
	if now.Second() >= 58 {
		time.Sleep(2 * time.Second)
		now = time.Now()
	}
	spec := fmt.Sprintf("%d,%d %d %d %d %d ?",
		now.Second()+1, now.Second()+2, now.Minute(), now.Hour(), now.Day(), now.Month())

	c := cron.New(cron.WithParser(cron.NewParser(cron.Second|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.DowOptional|cron.Descriptor)), cron.WithChain())
	c.AddFunc(spec, func() { wg.Done() })
	c.Start()
	defer c.Stop()

	select {
	case <-time.After(time.Second * 2):
		t.Error("expected job fires 2 times")
	case <-wait(wg):
	}
}
