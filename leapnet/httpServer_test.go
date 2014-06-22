/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

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

package leapnet

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/jeffail/leaps/leaplib"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

type binderStory struct {
	Content    string                `json:"content"`
	Transforms []*leaplib.OTransform `json:"transforms"`
	TCorrected []*leaplib.OTransform `json:"corrected_transforms"`
	Result     string                `json:"result"`
}

type binderStoriesContainer struct {
	Stories []binderStory `json:"binder_stories"`
}

func findDocument(id string, ws *websocket.Conn) error {
	websocket.JSON.Send(ws, LeapClientMessage{
		Command: "find",
		ID:      id,
	})

	var initResponse LeapServerMessage
	if err := websocket.JSON.Receive(ws, &initResponse); err != nil {
		return fmt.Errorf("Init receive error: %v", err)
	}

	switch initResponse.Type {
	case "document":
	case "error":
		return fmt.Errorf("Server returned error: %v", initResponse.Error)
	default:
		return fmt.Errorf("unexpected message type from server init: %v", initResponse.Type)
	}

	return nil
}

func senderClient(id string, feeds <-chan *leaplib.OTransform, t *testing.T) {
	origin := "http://localhost/"
	url := "ws://localhost:8787/leapsocket"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Errorf("client connect error: %v", err)
		return
	}

	if err = findDocument(id, ws); err != nil {
		t.Errorf("%v", err)
		return
	}

	rcvChan := make(chan []*leaplib.OTransform, 5)
	crctChan := make(chan bool)
	go func() {
		for {
			var serverMsg LeapServerMessage
			if err := websocket.JSON.Receive(ws, &serverMsg); err == nil {
				switch serverMsg.Type {
				case "correction":
					crctChan <- true
				case "transforms":
					rcvChan <- serverMsg.Transforms
				case "error":
					t.Errorf("Server returned error: %v", serverMsg.Error)
				default:
					t.Errorf("unexpected message type from server: %v", serverMsg.Type)
				}
			} else {
				close(rcvChan)
				return
			}
		}
	}()

	for {
		select {
		case feed, open := <-feeds:
			if !open {
				return
			}
			websocket.JSON.Send(ws, LeapClientMessage{
				Command:   "submit",
				Transform: feed,
			})
			select {
			case <-crctChan:
			case <-time.After(1 * time.Second):
				t.Errorf("Timed out waiting for correction")
				return
			}
		case _, open := <-rcvChan:
			if !open {
				return
			}
			t.Errorf("sender client received a transform")
		case <-time.After(8 * time.Second):
			t.Errorf("Sender client timeout occured")
			return
		}
	}
}
func goodStoryClient(id string, bstory *binderStory, wg *sync.WaitGroup, t *testing.T) {
	origin := "http://localhost/"
	url := "ws://localhost:8787/leapsocket"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		t.Errorf("client connect error: %v", err)
		wg.Done()
		return
	}

	if err = findDocument(id, ws); err != nil {
		t.Errorf("%v", err)
		wg.Done()
		return
	}

	rcvChan := make(chan []*leaplib.OTransform, 5)
	go func() {
		for {
			var serverMsg LeapServerMessage
			if err := websocket.JSON.Receive(ws, &serverMsg); err == nil {
				switch serverMsg.Type {
				case "correction":
					t.Errorf("listening client received correction")
				case "transforms":
					rcvChan <- serverMsg.Transforms
				case "error":
					t.Errorf("Server returned error: %v", serverMsg.Error)
				default:
					t.Errorf("unexpected message type from server: %v", serverMsg.Type)
				}
			} else {
				close(rcvChan)
				return
			}
		}
	}()

	tformIndex, lenCorrected := 0, len(bstory.TCorrected)
	for {
		select {
		case ret, open := <-rcvChan:
			if !open {
				t.Errorf("channel was closed before receiving last change")
				wg.Done()
				return
			}
			for _, tform := range ret {
				if tform.Version != bstory.TCorrected[tformIndex].Version ||
					tform.Insert != bstory.TCorrected[tformIndex].Insert ||
					tform.Delete != bstory.TCorrected[tformIndex].Delete ||
					tform.Position != bstory.TCorrected[tformIndex].Position {
					t.Errorf("Transform (%v) not expected, %v != %v",
						tformIndex, tform, bstory.TCorrected[tformIndex])
				}
				tformIndex++
				if tformIndex == lenCorrected {
					wg.Done()
					return
				}
			}
		case <-time.After(8 * time.Second):
			t.Errorf("Timeout occured")
			wg.Done()
			return
		}
	}
}

func TestHttpServer(t *testing.T) {
	bytes, err := ioutil.ReadFile("../data/binder_stories.js")
	if err != nil {
		t.Errorf("Read file error: %v", err)
		return
	}

	var scont binderStoriesContainer
	if err := json.Unmarshal(bytes, &scont); err != nil {
		t.Errorf("Story parse error: %v", err)
		return
	}

	curatorConfig := leaplib.DefaultCuratorConfig()
	curatorConfig.BinderConfig.LogVerbose = false

	httpServerConfig := DefaultHTTPServerConfig()
	httpServerConfig.LogVerbose = false
	httpServerConfig.URL.Address = ":8787"

	curator, err := leaplib.CreateNewCurator(curatorConfig)
	if err != nil {
		t.Errorf("Curator error: %v", err)
	}

	go func() {
		http, err := CreateHTTPServer(curator, httpServerConfig)
		if err != nil {
			t.Errorf("Create HTTP error: %v", err)
			return
		}
		if err = http.Listen(); err != nil {
			t.Errorf("Listen error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	origin := "http://localhost/"
	url := "ws://localhost:8787/leapsocket"

	for i, story := range scont.Stories {

		ws, err := websocket.Dial(url, "", origin)
		if err != nil {
			t.Errorf("client connect error: %v", err)
			return
		}

		websocket.JSON.Send(ws, LeapClientMessage{
			Command: "create",
			Document: &leaplib.Document{
				Title:       fmt.Sprintf("Story%v", i),
				Description: fmt.Sprintf("Story #%v", i),
				Content:     story.Content,
			},
		})

		var initResponse LeapServerMessage
		if err := websocket.JSON.Receive(ws, &initResponse); err != nil {
			t.Errorf("Init receive error: %v", err)
			return
		}

		switch initResponse.Type {
		case "document":
		case "error":
			t.Errorf("Server returned error: %v", initResponse.Error)
			return
		default:
			t.Errorf("unexpected message type from server init: %v", initResponse.Type)
			return
		}

		feeds := make(chan *leaplib.OTransform)

		wg := sync.WaitGroup{}
		wg.Add(10)

		go senderClient(initResponse.Document.ID, feeds, t)
		for j := 0; j < 10; j++ {
			go goodStoryClient(initResponse.Document.ID, &story, &wg, t)
		}

		time.Sleep(1 * time.Second)

		for j := 0; j < len(story.Transforms); j++ {
			feeds <- story.Transforms[j]
		}

		wg.Wait()
	}

	curator.Close()
}
