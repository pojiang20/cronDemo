package main

import (
	"fmt"
	"github.com/robfig/cron"
	"time"
)

func main() {
	c := cron.New()

	c.AddFunc("@every 1s", func() {
		fmt.Println("now: ", time.Now().String())
	})

	c.Start()
	time.Sleep(time.Second * 5)
}
