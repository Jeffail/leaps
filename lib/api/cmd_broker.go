/*
Copyright (c) 2017 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, sub to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package api

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

//------------------------------------------------------------------------------

// CMDRunner is a type that executes commands, returning separate stdout/stderr
// results as well as an error.
type CMDRunner interface {
	// CMDRun runs the provided command in a shell and returns the result.
	CMDRun(cmd string) (stdout, stderr []byte, err error)
}

//------------------------------------------------------------------------------

// CMDBroker - The leaps API allows the service owner to specify a static list
// of available commands. The broker is responsible for providing each new
// client the list of commands, then listens for requests by the client to run
// those commands. When a command is run the broker broadcasts the output to all
// connected clients.
type CMDBroker struct {
	emitters []Emitter

	cmds      []string
	cmdRunner CMDRunner
	cmdChan   chan int

	timeout time.Duration
	logger  log.Modular
	stats   metrics.Type

	emittersMut sync.Mutex
	closeChan   chan struct{}
}

// NewCMDBroker - Create a new instance of an event broker.
func NewCMDBroker(
	cmds []string,
	cmdRunner CMDRunner,
	timeout time.Duration,
	logger log.Modular,
	stats metrics.Type,
) *CMDBroker {
	b := &CMDBroker{
		cmds:      cmds,
		cmdRunner: cmdRunner,
		cmdChan:   make(chan int),
		timeout:   timeout,
		logger:    logger.NewModule(":api:cmd_broker"),
		stats:     stats,
		closeChan: make(chan struct{}),
	}
	go b.cmdLoop()
	return b
}

//------------------------------------------------------------------------------

func (b *CMDBroker) cmdLoop() {
	for {
		select {
		case <-b.closeChan:
			return
		case cmd, open := <-b.cmdChan:
			if !open {
				return
			}
			if cmd >= 0 && cmd < len(b.cmds) {
				stdout, stderr, err := b.cmdRunner.CMDRun(b.cmds[cmd])
				errStr := ""
				if err != nil {
					errStr = err.Error()
				}

				b.dispatch(nil, events.GlobalMetadata, events.GlobalMetadataMessage{
					Metadata: events.MetadataBody{
						Type: events.CMDOutput,
						Body: events.CMDMetadataMessage{
							CMDData: events.CMDData{
								ID:     cmd,
								Error:  errStr,
								Stdout: string(stdout),
								Stderr: string(stderr),
							},
						},
					},
				})
			}
		}
	}
}

//------------------------------------------------------------------------------

// NewEmitter - Register a new emitter to the broker, the emitter will begin
// receiving globally broadcast events from other emitters.
func (b *CMDBroker) NewEmitter(username, uuid string, e Emitter) {
	e.OnClose(func() {
		b.emittersMut.Lock()
		if len(b.emitters) > 0 {
			var i int
			for i = range b.emitters {
				if b.emitters[i] == e {
					break
				}
			}
			// Just incase
			if b.emitters[i] == e {
				b.stats.Decr("api.cmd_broker.emitters", 1)
				b.emitters = append(b.emitters[:i], b.emitters[i+1:]...)
			}
		}
		b.emittersMut.Unlock()
	})
	e.OnReceive(events.GlobalMetadata, func(body []byte) events.TypedError {
		var cmdReq events.CMDMetadataMessage
		var metBody events.MetadataBody
		var req events.GlobalMetadataMessage
		metBody.Body = &cmdReq
		req.Metadata = &metBody
		if err := json.Unmarshal(body, &req); err != nil {
			b.stats.Incr("api.cmd_broker.metadata.error.json", 1)
			b.logger.Warnf("Message parse error: %v\n", err)
			return events.NewAPIError(events.ErrBadJSON, err.Error())
		}
		if metBody.Type == events.CMD {
			if cmdReq.CMDData.ID >= 0 && cmdReq.CMDData.ID < len(b.cmds) {
				select {
				case b.cmdChan <- cmdReq.CMDData.ID:
				case <-time.After(b.timeout):
				}
			} else {
				b.stats.Incr("api.cmd_broker.metadata.error.oob", 1)
				b.logger.Warnf("CMD Index out of bounds: %v\n", cmdReq.CMDData.ID)
				return events.NewAPIError(events.ErrBadReq, "CMD Index was out of bounds")
			}
		}
		return nil
	})

	b.emittersMut.Lock()
	b.emitters = append(b.emitters, e)
	b.emittersMut.Unlock()

	// Send list of available cmds to new client
	e.Send(events.GlobalMetadata, events.GlobalMetadataMessage{
		Metadata: events.MetadataBody{
			Type: events.CMDList,
			Body: events.CMDListMetadataMessage{
				CMDS: b.cmds,
			},
		},
	})

	b.stats.Incr("api.cmd_broker.emitters", 1)
}

// Close initiates the shut down of the background cmd runner.
func (b *CMDBroker) Close(time.Duration) {
	close(b.closeChan)
}

// dispatch - Dispatch an event to all active emitters, this is done
// synchronously with the trigger event in order to avoid flooding.
func (b *CMDBroker) dispatch(source Emitter, typeStr string, body interface{}) {
	b.emittersMut.Lock()
	activeEmitters := b.emitters[:]

	wg := sync.WaitGroup{}
	wg.Add(len(activeEmitters))
	for _, e := range activeEmitters {
		go func(em Emitter) {
			if em != source {
				em.Send(typeStr, body)
			}
			wg.Done()
		}(e)
	}
	wg.Wait()
	b.emittersMut.Unlock()
}

//------------------------------------------------------------------------------
