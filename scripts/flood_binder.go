package main

import (
	"fmt"
	ll "github.com/jeffail/leaps/leaplib"
	"time"
)

func main() {
	errChan := make(chan ll.BinderError)
	doc, err := ll.CreateNewDocument("test", "test1", "text", "hello world 123")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	logConf := ll.DefaultLoggerConfig()
	//logConf.LogLevel = ll.LeapDebug

	logger := ll.CreateLogger(logConf)

	store, _ := ll.GetMemoryStore(ll.DocumentStoreConfig{})

	binderConfig := ll.DefaultBinderConfig()
	binderConfig.RetentionPeriod = 1

	binder, err := ll.BindNew(doc, store, binderConfig, errChan, logger)
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
			ll.OTransform{
				Position: 0,
				Version:  targetV,
				Delete:   11,
				Insert:   "hello world",
			},
			time.Second,
		); v != targetV || err != nil {
			fmt.Printf("Send Transform error, targetV: %v, v: %v, err: %v\n", targetV, v, err)
		}
		targetV++
	}
}
