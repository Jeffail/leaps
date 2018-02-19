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
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
)

//------------------------------------------------------------------------------

func TestBasicGlobalMetadataBroker(t *testing.T) {
	eBroker := NewGlobalMetadataBroker(time.Second, logger, stats)

	dEmitter1 := &dudEmitter{
		reqHandlers: map[string]RequestHandler{},
		resHandlers: map[string]ResponseHandler{},
		sendChan:    make(chan dudSendType),
	}

	go eBroker.NewEmitter("foo1", "bar1", dEmitter1)

	if err := compareGlobalMetadata(dEmitter1, events.GlobalMetadataMessage{
		Client: events.Client{Username: "foo1", SessionID: "bar1"},
		Metadata: events.MetadataBody{
			Type: events.UserInfo,
			Body: events.UserInfoMetadataMessage{
				Users: map[string]events.UserSubscriptions{
					"bar1": {Username: "foo1", Subscriptions: nil},
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}

	if _, ok := dEmitter1.reqHandlers[events.GlobalMetadata]; !ok {
		t.Error("Global metadata handler was not bound")
	}
	if dEmitter1.closeHandler == nil {
		t.Error("Close handler was not bound")
	}
	if exp, act := 1, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong tally of connected users: %v != %v", exp, act)
	}

	if exp, act := 1, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}

	// Ensure broker does not echo metadata
	dEmitter1.reqHandlers[events.GlobalMetadata]([]byte("{}"))

	select {
	case <-dEmitter1.sendChan:
		t.Error("Emitter received its own metadata event")
	case <-time.After(time.Millisecond * 100):
	}

	// Ensure clean up
	dEmitter1.closeHandler()

	if exp, act := 0, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}
	if exp, act := 0, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong tally of connected users: %v != %v", exp, act)
	}
}

func TestStressedGlobalMetadataBroker(t *testing.T) {
	eBroker := NewGlobalMetadataBroker(time.Second, logger, stats)

	dEmitterOG := &dudEmitter{
		reqHandlers: map[string]RequestHandler{},
		resHandlers: map[string]ResponseHandler{},
		sendChan:    make(chan dudSendType),
	}
	dEmitterLG := &dudEmitter{
		reqHandlers: map[string]RequestHandler{},
		resHandlers: map[string]ResponseHandler{},
		sendChan:    make(chan dudSendType),
	}

	go eBroker.NewEmitter("OG", "OG", dEmitterOG)
	select {
	case <-dEmitterOG.sendChan:
	case <-time.After(time.Second):
		t.Error("Timed out waiting for OG to get user_info message")
	}

	wg := sync.WaitGroup{}
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			em := &dudEmitter{
				reqHandlers: map[string]RequestHandler{},
				resHandlers: map[string]ResponseHandler{},
			}
			go func() {
				select {
				case <-dEmitterOG.sendChan:
				case <-time.After(time.Second):
					t.Error("Timed out waiting for OG to get connect message")
				}
			}()
			eBroker.NewEmitter("foo", "bar", em)
			<-time.After(time.Millisecond * 10)
			go em.closeHandler()
			select {
			case <-dEmitterOG.sendChan:
			case <-time.After(time.Second):
				t.Error("Timed out waiting for OG to get disconnect message")
			}
			wg.Done()
		}()
	}

	wg.Wait()

	if exp, act := 1, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}
	if exp, act := 1, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong count of users: %v != %v", exp, act)
	}

	go eBroker.NewEmitter("LG", "LG", dEmitterLG)
	select {
	case <-dEmitterLG.sendChan:
	case <-time.After(time.Second):
		t.Error("Timed out waiting for LG to get user_info message")
	}

	if err := compareGlobalMetadata(dEmitterOG, events.GlobalMetadataMessage{
		Client: events.Client{Username: "LG", SessionID: "LG"},
		Metadata: events.MetadataBody{
			Type: events.UserConnect,
			Body: nil,
		},
	}); err != nil {
		t.Error(err)
	}

	if exp, act := 2, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}
	if exp, act := 2, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong count of users: %v != %v", exp, act)
	}

	go dEmitterOG.reqHandlers[events.GlobalMetadata]([]byte("{}"))

	if err := compareGlobalMetadata(dEmitterLG, events.GlobalMetadataMessage{
		Client:   events.Client{Username: "OG", SessionID: "OG"},
		Metadata: nil,
	}); err != nil {
		t.Error(err)
	}

	go dEmitterOG.closeHandler()

	if err := compareGlobalMetadata(dEmitterLG, events.GlobalMetadataMessage{
		Client: events.Client{Username: "OG", SessionID: "OG"},
		Metadata: events.MetadataBody{
			Type: events.UserDisconnect,
			Body: nil,
		},
	}); err != nil {
		t.Error(err)
	}

	if exp, act := 1, len(eBroker.emitters); exp != act {
		t.Errorf("Wrong count of emitters: %v != %v", exp, act)
	}
	if exp, act := 1, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong count of users: %v != %v", exp, act)
	}
}

func TestSubTrackingGlobalMetadataBroker(t *testing.T) {
	targetSubscriptions := map[string]events.UserSubscriptions{
		"bar1": {
			Username:      "foo1",
			Subscriptions: []string{"first"},
		},
		"bar2": {
			Username:      "foo2",
			Subscriptions: []string{"second", "third"},
		},
		"bar3": {
			Username:      "foo3",
			Subscriptions: []string{"second", "third", "fourth"},
		},
		"bar4": {
			Username:      "foo4",
			Subscriptions: []string{"first", "fourth", "fifth"},
		},
	}

	expectedSubscriptions := map[string]events.UserSubscriptions{}

	eBroker := NewGlobalMetadataBroker(time.Second, logger, stats)
	emitters := map[string]*dudEmitter{}

	// Do subscriptions
	for k, v := range targetSubscriptions {
		expectedSubscriptions[k] = events.UserSubscriptions{
			Username: v.Username, Subscriptions: nil,
		}

		newEmitter := &dudEmitter{
			reqHandlers: map[string]RequestHandler{},
			resHandlers: map[string]ResponseHandler{},
			sendChan:    make(chan dudSendType),
		}

		// On a separate thread we simulate a new socket connection and
		// subsequent subscription calls.
		wg := sync.WaitGroup{}
		wg.Add(2 + len(emitters))

		go func(em *dudEmitter, subs []string) {
			go func() {
				eBroker.NewEmitter(v.Username, k, em)
				wg.Done()
			}()
			if err := compareGlobalMetadata(em, events.GlobalMetadataMessage{
				Client: events.Client{Username: v.Username, SessionID: k},
				Metadata: events.MetadataBody{
					Type: events.UserInfo,
					Body: events.UserInfoMetadataMessage{
						Users: expectedSubscriptions,
					},
				},
			}); err != nil {
				t.Error(err)
			}
			for _, sub := range subs {
				em.resHandlers[events.Subscribe](
					events.SubscriptionMessage{
						Document: events.DocumentFull{ID: sub},
					},
				)
			}
			wg.Done()
		}(newEmitter, v.Subscriptions)

		for _, emitter := range emitters {
			go func(em *dudEmitter) {
				if err := compareGlobalMetadata(em, events.GlobalMetadataMessage{
					Client:   events.Client{Username: v.Username, SessionID: k},
					Metadata: events.MetadataBody{Type: events.UserConnect, Body: nil},
				}); err != nil {
					t.Error(err)
				}
				for _, sub := range v.Subscriptions {
					if err := compareGlobalMetadata(em, events.GlobalMetadataMessage{
						Client: events.Client{Username: v.Username, SessionID: k},
						Metadata: events.MetadataBody{
							Type: events.UserSubscribe,
							Body: events.UnsubscriptionMessage{
								Document: events.DocumentStripped{ID: sub},
							},
						},
					}); err != nil {
						t.Error(err)
					}
				}
				wg.Done()
			}(emitter)
		}

		wg.Wait()

		emitters[k] = newEmitter
		expectedSubscriptions[k] = v
	}

	if exp, act := len(targetSubscriptions), len(eBroker.userMap); exp != act {
		t.Errorf("Wrong final count of subs: %v != %v", exp, act)
	}

	i := 0
	for k, v := range targetSubscriptions {
		wg := sync.WaitGroup{}
		wg.Add(len(emitters))

		go func(em *dudEmitter, subs []string) {
			if i%2 == 0 {
				for _, sub := range subs {
					em.resHandlers[events.Unsubscribe](
						events.UnsubscriptionMessage{
							Document: events.DocumentStripped{ID: sub},
						},
					)
				}
			}
			em.closeHandler()
			wg.Done()
		}(emitters[k], v.Subscriptions)

		delete(emitters, k)

		for _, remainingEmitter := range emitters {
			go func(em *dudEmitter, subs []string) {
				if i%2 == 0 {
					for _, sub := range subs {
						if err := compareGlobalMetadata(em, events.GlobalMetadataMessage{
							Client: events.Client{Username: v.Username, SessionID: k},
							Metadata: events.MetadataBody{
								Type: events.UserUnsubscribe,
								Body: events.UnsubscriptionMessage{
									Document: events.DocumentStripped{ID: sub},
								},
							},
						}); err != nil {
							t.Error(err)
						}
					}
				}
				if err := compareGlobalMetadata(em, events.GlobalMetadataMessage{
					Client:   events.Client{Username: v.Username, SessionID: k},
					Metadata: events.MetadataBody{Type: events.UserDisconnect, Body: nil},
				}); err != nil {
					t.Error(err)
				}
				wg.Done()
			}(remainingEmitter, v.Subscriptions)
		}

		wg.Wait()
		i++
	}

	if exp, act := 0, len(eBroker.userMap); exp != act {
		t.Errorf("Wrong final count of subs: %v != %v", exp, act)
	}
}

func TestSingleTrackingGlobalMetadataBroker(t *testing.T) {
	expectedSubscriptions := map[string]events.UserSubscriptions{
		"bar1": {
			Username:      "foo1",
			Subscriptions: nil,
		},
	}

	eBroker := NewGlobalMetadataBroker(time.Second, logger, stats)
	emitter := &dudEmitter{
		reqHandlers: map[string]RequestHandler{},
		resHandlers: map[string]ResponseHandler{},
		sendChan:    make(chan dudSendType),
	}

	go eBroker.NewEmitter("foo1", "bar1", emitter)
	if err := compareGlobalMetadata(emitter, events.GlobalMetadataMessage{
		Client: events.Client{Username: "foo1", SessionID: "bar1"},
		Metadata: events.MetadataBody{
			Type: events.UserInfo,
			Body: events.UserInfoMetadataMessage{
				Users: expectedSubscriptions,
			},
		},
	}); err != nil {
		t.Error(err)
	}

	emitter.resHandlers[events.Subscribe](
		events.SubscriptionMessage{
			Document: events.DocumentFull{ID: "doc1"},
		},
	)

	expectedSubscriptions["bar1"] = events.UserSubscriptions{
		Username:      "foo1",
		Subscriptions: []string{"doc1"},
	}
	if !reflect.DeepEqual(expectedSubscriptions, eBroker.userMap) {
		t.Errorf("Wrong user subs: %v != %v", expectedSubscriptions, eBroker.userMap)
	}

	emitter.resHandlers[events.Subscribe](
		events.SubscriptionMessage{
			Document: events.DocumentFull{ID: "doc2"},
		},
	)

	expectedSubscriptions["bar1"] = events.UserSubscriptions{
		Username:      "foo1",
		Subscriptions: []string{"doc1", "doc2"},
	}
	if !reflect.DeepEqual(expectedSubscriptions, eBroker.userMap) {
		t.Errorf("Wrong user subs: %v != %v", expectedSubscriptions, eBroker.userMap)
	}

	emitter.resHandlers[events.Unsubscribe](
		events.UnsubscriptionMessage{
			Document: events.DocumentStripped{ID: "doc1"},
		},
	)

	expectedSubscriptions["bar1"] = events.UserSubscriptions{
		Username:      "foo1",
		Subscriptions: []string{"doc2"},
	}
	if !reflect.DeepEqual(expectedSubscriptions, eBroker.userMap) {
		t.Errorf("Wrong user subs: %v != %v", expectedSubscriptions, eBroker.userMap)
	}

	emitter.resHandlers[events.Unsubscribe](
		events.UnsubscriptionMessage{
			Document: events.DocumentStripped{ID: "doc2"},
		},
	)

	expectedSubscriptions["bar1"] = events.UserSubscriptions{
		Username:      "foo1",
		Subscriptions: []string{},
	}
	if !reflect.DeepEqual(expectedSubscriptions, eBroker.userMap) {
		t.Errorf("Wrong user subs: %v != %v", expectedSubscriptions, eBroker.userMap)
	}
}

//------------------------------------------------------------------------------
