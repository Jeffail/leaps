package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jeffail/leaps/lib"
	"github.com/jeffail/leaps/util"
)

func main() {
	logConf := util.DefaultLoggerConfig()
	logConf.LogLevel = "INFO"

	logger := util.NewLogger(os.Stdout, logConf)
	stats := util.NewStats(util.DefaultStatsConfig())

	errChan := make(chan lib.BinderError)
	doc, err := lib.NewDocument("helibo world 123")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	store, _ := lib.GetMemoryStore(lib.DocumentStoreConfig{})

	binderConfig := lib.DefaultBinderConfig()
	binderConfig.RetentionPeriod = 1

	if err := store.Create(doc.ID, doc); err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	binder, err := lib.NewBinder(doc.ID, store, binderConfig, errChan, logger, stats)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	go func() {
		for err := range errChan {
			fmt.Printf("From error channel: %v\n", err.Err)
		}
	}()

	portal, portal2 := binder.Subscribe(""), binder.Subscribe("")

	go func() {
		for _ = range portal2.TransformRcvChan {
		}
	}()

	time.Sleep(time.Second * 2)
	targetV := 2

	for {
		if v, err := portal.SendTransform(
			lib.OTransform{
				Position: 0,
				Version:  targetV,
				Delete:   11,
				Insert:   "helibo world",
			},
			time.Second,
		); v != targetV || err != nil {
			fmt.Printf("Send Transform error, targetV: %v, v: %v, err: %v\n", targetV, v, err)
		}
		targetV++
	}
}
