package main

import (
	"fmt"
	"time"

	"github.com/jeffail/leaps/lib"
)

func main() {
	errChan := make(chan lib.BinderError)
	doc, err := lib.CreateNewDocument("test", "test1", "text", "helibo world 123")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	logConf := lib.DefaultLoggerConfig()
	//logConf.LogLevel = lib.LeapDebug

	logger := lib.CreateLogger(logConf)

	store, _ := lib.GetMemoryStore(lib.DocumentStoreConfig{})

	binderConfig := lib.DefaultBinderConfig()
	binderConfig.RetentionPeriod = 1

	binder, err := lib.BindNew(doc, store, binderConfig, errChan, logger)
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
