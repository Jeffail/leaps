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

// GlobalMetadataBroker - The leaps API defines events that are outside of the
// functionality of curators and binders, such as chat messages and user
// join/leave notifications. The GlobalMetadataBroker implements these functions
// by managing references to open Emitters.
type GlobalMetadataBroker struct {
	emitters []Emitter
	userMap  map[string]events.UserSubscriptions

	timeout time.Duration
	logger  log.Modular
	stats   metrics.Type

	userMapMut  sync.Mutex
	emittersMut sync.Mutex
}

// NewGlobalMetadataBroker - Create a new instance of an event broker.
func NewGlobalMetadataBroker(
	timeout time.Duration,
	logger log.Modular,
	stats metrics.Type,
) *GlobalMetadataBroker {
	return &GlobalMetadataBroker{
		userMap: make(map[string]events.UserSubscriptions),
		timeout: timeout,
		logger:  logger.NewModule(":api:global_metadata_broker"),
		stats:   stats,
	}
}

//------------------------------------------------------------------------------

// NewEmitter - Register a new emitter to the broker, the emitter will begin
// receiving globally broadcast events from other emitters.
func (b *GlobalMetadataBroker) NewEmitter(username, uuid string, e Emitter) {
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
				b.stats.Decr("api.global_metadata_broker.emitters", 1)
				b.emitters = append(b.emitters[:i], b.emitters[i+1:]...)
			}
		}
		b.emittersMut.Unlock()

		b.userMapMut.Lock()
		delete(b.userMap, uuid)
		b.userMapMut.Unlock()

		b.dispatch(e, events.GlobalMetadata, events.GlobalMetadataMessage{
			Client:   events.Client{Username: username, SessionID: uuid},
			Metadata: events.MetadataBody{Type: events.UserDisconnect, Body: nil},
		})
	})
	e.OnSend(events.Subscribe, func(body interface{}) bool {
		subBody, ok := body.(events.SubscriptionMessage)
		if !ok {
			return true
		}

		// Update the internal map
		b.userMapMut.Lock()
		userData := b.userMap[uuid]
		userData.Subscriptions = append(userData.Subscriptions, subBody.Document.ID)
		b.userMap[uuid] = userData
		b.userMapMut.Unlock()

		b.dispatch(e, events.GlobalMetadata, events.GlobalMetadataMessage{
			Client: events.Client{Username: username, SessionID: uuid},
			Metadata: events.MetadataBody{
				Type: events.UserSubscribe,
				Body: events.UnsubscriptionMessage{
					Document: events.DocumentStripped{
						ID: subBody.Document.ID,
					},
				},
			},
		})
		return true
	})
	e.OnSend(events.Unsubscribe, func(body interface{}) bool {
		subBody, ok := body.(events.UnsubscriptionMessage)
		if !ok {
			return true
		}

		// Update the internal map
		b.userMapMut.Lock()
		if userData, ok := b.userMap[uuid]; ok {
			i := 0
			for _, sub := range userData.Subscriptions {
				if sub != subBody.Document.ID {
					userData.Subscriptions[i] = sub
					i++
				}
			}
			userData.Subscriptions = userData.Subscriptions[:i]
			b.userMap[uuid] = userData
		}
		b.userMapMut.Unlock()

		b.dispatch(e, events.GlobalMetadata, events.GlobalMetadataMessage{
			Client: events.Client{Username: username, SessionID: uuid},
			Metadata: events.MetadataBody{
				Type: events.UserUnsubscribe,
				Body: events.UnsubscriptionMessage{
					Document: events.DocumentStripped{
						ID: subBody.Document.ID,
					},
				},
			},
		})
		return true
	})
	e.OnReceive(events.GlobalMetadata, func(body []byte) events.TypedError {
		var req events.GlobalMetadataMessage
		if err := json.Unmarshal(body, &req); err != nil {
			b.stats.Incr("api.global_metadata_broker.metadata.error.json", 1)
			b.logger.Warnf("Message parse error: %v\n", err)
			return events.NewAPIError(events.ErrBadJSON, err.Error())
		}
		b.dispatch(e, events.GlobalMetadata, events.GlobalMetadataMessage{
			Client:   events.Client{Username: username, SessionID: uuid},
			Metadata: req.Metadata,
		})
		return nil
	})

	b.emittersMut.Lock()
	b.emitters = append(b.emitters, e)
	b.emittersMut.Unlock()

	// Create safe copy of user subscriptions map
	usersCopy := map[string]events.UserSubscriptions{}

	b.userMapMut.Lock()
	b.userMap[uuid] = events.UserSubscriptions{Username: username, Subscriptions: nil}
	for k, v := range b.userMap {
		usersCopy[k] = v
	}
	b.userMapMut.Unlock()

	// Send welcome UserInfo message
	e.Send(events.GlobalMetadata, events.GlobalMetadataMessage{
		Client: events.Client{Username: username, SessionID: uuid},
		Metadata: events.MetadataBody{
			Type: events.UserInfo,
			Body: events.UserInfoMetadataMessage{Users: usersCopy},
		},
	})

	// Dispatch UserConnect to other clients
	b.dispatch(e, events.GlobalMetadata, events.GlobalMetadataMessage{
		Client:   events.Client{Username: username, SessionID: uuid},
		Metadata: events.MetadataBody{Type: events.UserConnect, Body: nil},
	})

	b.stats.Incr("api.global_metadata_broker.emitters", 1)
}

// dispatch - Dispatch an event to all active emitters, this is done
// synchronously with the trigger event in order to avoid flooding.
func (b *GlobalMetadataBroker) dispatch(source Emitter, typeStr string, body interface{}) {
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
