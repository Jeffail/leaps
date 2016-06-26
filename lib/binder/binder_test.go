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

package binder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/text"
	"github.com/jeffail/leaps/lib/util"
)

//--------------------------------------------------------------------------------------------------

// testStore - Just stores documents in a map.
type testStore struct {
	documents map[string]store.Document
	mutex     sync.RWMutex
}

// Create - Store document in memory.
func (s *testStore) Create(doc store.Document) error {
	return s.Update(doc)
}

// Update - Store document in memory.
func (s *testStore) Update(doc store.Document) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.documents[doc.ID] = doc
	return nil
}

// Read - Fetch document from memory.
func (s *testStore) Read(id string) (store.Document, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	doc, ok := s.documents[id]
	if !ok {
		return doc, store.ErrDocumentNotExist
	}
	return doc, nil
}

//--------------------------------------------------------------------------------------------------

func TestGracefullShutdown(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc, _ := store.NewDocument("hello world")

	store := testStore{documents: map[string]store.Document{
		"KILL_ME": *doc,
	}}

	binder, err := New("KILL_ME", &store, NewConfig(), errChan, logger, stats)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	testClient, err := binder.Subscribe("", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	delete(store.documents, "KILL_ME")
	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 2}, time.Second)

	<-errChan
	binder.Close()
}

func TestClientAdminTasks(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc, _ := store.NewDocument("hello world")

	store := testStore{documents: map[string]store.Document{
		"KILL_ME": *doc,
	}}

	binder, err := New("KILL_ME", &store, NewConfig(), errChan, logger, stats)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	nClients := 10

	portals := make([]Portal, nClients)
	clientIDs := make([]string, nClients)

	for i := 0; i < nClients; i++ {
		clientIDs[i] = util.GenerateStampedUUID()
		var err error
		portals[i], err = binder.Subscribe(clientIDs[i], time.Second)
		if err != nil {
			t.Errorf("Subscribe error: %v\n", err)
			return
		}
	}

	for i := 0; i < nClients; i++ {
		remainingClients, err := binder.GetUsers(time.Second)
		if err != nil {
			t.Errorf("Get users error: %v\n", err)
			return
		}
		if len(remainingClients) != len(clientIDs) {
			t.Errorf("Wrong number of remaining clients: %v != %v\n", len(remainingClients), len(clientIDs))
			return
		}
		for _, val := range clientIDs {
			found := false
			for _, c := range remainingClients {
				if val == c {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Client not found in binder: %v\n", val)
				return
			}
		}

		killID := clientIDs[0]
		clientIDs = clientIDs[1:]

		if err := binder.KickUser(killID, time.Second); err != nil {
			t.Errorf("Kick user error: %v\n", err)
			return
		}
	}

	binder.Close()
}

func TestKickLockedUsers(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc, _ := store.NewDocument("hello world")

	store := testStore{documents: map[string]store.Document{
		"KILL_ME": *doc,
	}}

	conf := NewConfig()
	conf.ClientKickPeriod = 1

	binder, err := New("KILL_ME", &store, conf, errChan, logger, stats)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	defer binder.Close()

	testClient, err := binder.Subscribe("TestClient", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = binder.Subscribe("TestClient2", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = binder.Subscribe("TestClient3", time.Second)
	if err != nil {
		t.Error(err)
		return
	}

	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 2}, time.Second)
	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 3}, time.Second)
	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 4}, time.Second)

	// Wait until both testClient2 and testClient3 should have been kicked (in the same epoch).
	<-time.After(time.Millisecond * 10)
}

func TestUpdates(t *testing.T) {
	errChan := make(chan Error)
	doc, _ := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: *doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
	)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	go func() {
		for e := range errChan {
			t.Errorf("From error channel: %v", e.Err)
		}
	}()

	userID1, userID2 := util.GenerateStampedUUID(), util.GenerateStampedUUID()

	portal1, err := binder.Subscribe(userID1, time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	portal2, err := binder.Subscribe(userID2, time.Second)
	if err != nil {
		t.Error(err)
		return
	}

	if userID1 != portal1.UserID() {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID1, portal1.UserID())
	}
	if userID2 != portal2.UserID() {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID2, portal2.UserID())
	}

	for i := 0; i < 100; i++ {
		portal1.SendMessage(Message{})

		message := <-portal2.UpdateReadChan()
		if message.ClientInfo.UserID != userID1 {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message.ClientInfo.UserID, portal1.UserID(),
			)
		}
		if message.ClientInfo.SessionID != portal1.SessionID() {
			t.Errorf(
				"Received incorrect session ID: %v != %v",
				message.ClientInfo.SessionID, portal1.SessionID(),
			)
		}

		portal2.SendMessage(Message{})

		message2 := <-portal1.UpdateReadChan()
		if message2.ClientInfo.UserID != userID2 {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message2.ClientInfo.UserID, portal2.UserID(),
			)
		}
		if message2.ClientInfo.SessionID != portal2.SessionID() {
			t.Errorf(
				"Received incorrect session ID: %v != %v",
				message2.ClientInfo.SessionID, portal2.SessionID())
		}
	}
}

func TestUpdatesSameUserID(t *testing.T) {
	errChan := make(chan Error)
	doc, _ := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: *doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
	)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	go func() {
		for e := range errChan {
			t.Errorf("From error channel: %v", e.Err)
		}
	}()

	userID := util.GenerateStampedUUID()

	portal1, _ := binder.Subscribe(userID, time.Second)
	portal2, _ := binder.Subscribe(userID, time.Second)

	if userID != portal1.UserID() {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID, portal1.UserID())
	}
	if userID != portal2.UserID() {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID, portal2.UserID())
	}

	for i := 0; i < 100; i++ {
		portal1.SendMessage(Message{})

		message := <-portal2.UpdateReadChan()
		if message.ClientInfo.UserID != userID {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message.ClientInfo.UserID, portal1.UserID(),
			)
		}
		if message.ClientInfo.SessionID != portal1.SessionID() {
			t.Errorf(
				"Received incorrect session ID: %v != %v",
				message.ClientInfo.SessionID, portal1.SessionID(),
			)
		}

		portal2.SendMessage(Message{})

		message2 := <-portal1.UpdateReadChan()
		if message2.ClientInfo.UserID != userID {
			t.Errorf(
				"Received incorrect token: %v != %v",
				message2.ClientInfo.UserID, portal2.UserID(),
			)
		}
		if message2.ClientInfo.SessionID != portal2.SessionID() {
			t.Errorf(
				"Received incorrect session ID: %v != %v",
				message2.ClientInfo.SessionID, portal2.SessionID(),
			)
		}
	}
}

func TestNew(t *testing.T) {
	errChan := make(chan Error)
	doc, _ := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: *doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
	)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	go func() {
		for err := range errChan {
			t.Errorf("From error channel: %v", err.Err)
		}
	}()

	portal1, _ := binder.Subscribe("", time.Second)
	portal2, _ := binder.Subscribe("", time.Second)
	if v, err := portal1.SendTransform(
		text.OTransform{
			Position: 6,
			Version:  2,
			Delete:   5,
			Insert:   "universe",
		},
		time.Second,
	); v != 2 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}
	if v, err := portal2.SendTransform(
		text.OTransform{
			Position: 0,
			Version:  3,
			Delete:   0,
			Insert:   "super ",
		},
		time.Second,
	); v != 3 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}

	<-portal1.TransformReadChan()
	<-portal2.TransformReadChan()

	portal3, _ := binder.Subscribe("", time.Second)
	if exp, rec := "super hello universe", portal3.Document().Content; exp != rec {
		t.Errorf("Wrong content, expected %v, received %v", exp, rec)
	}
}

func TestReadOnlyPortals(t *testing.T) {
	errChan := make(chan Error)
	doc, _ := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: *doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
	)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	go func() {
		for err := range errChan {
			t.Errorf("From error channel: %v", err.Err)
		}
	}()

	portal1, _ := binder.Subscribe("", time.Second)
	portal2, _ := binder.Subscribe("", time.Second)
	portalReadOnly, _ := binder.SubscribeReadOnly("", time.Second)

	if v, err := portal1.SendTransform(
		text.OTransform{
			Position: 6,
			Version:  2,
			Delete:   5,
			Insert:   "universe",
		},
		time.Second,
	); v != 2 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}
	if v, err := portal2.SendTransform(
		text.OTransform{
			Position: 0,
			Version:  3,
			Delete:   0,
			Insert:   "super ",
		},
		time.Second,
	); v != 3 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}

	<-portal1.TransformReadChan()
	<-portal2.TransformReadChan()
	<-portalReadOnly.TransformReadChan()

	if _, err := portalReadOnly.SendTransform(text.OTransform{}, time.Second); err != ErrReadOnlyPortal {
		t.Errorf("Read only portal unexpected result: %v", err)
	}

	portal3, _ := binder.Subscribe("", time.Second)
	if exp, rec := "super hello universe", portal3.Document().Content; exp != rec {
		t.Errorf("Wrong content, expected %v, received %v", exp, rec)
	}
}

/*func badClient(b *BinderPortal, t *testing.T, wg *sync.WaitGroup) {
	// Do nothing, LOLOLOLOLOL AHAHAHAHAHAHAHAHAHA! TIME WASTTTTIIINNNGGGG!!!!
	time.Sleep(500 * time.Millisecond)

	// The first transform is free (buffered chan)
	<-b.TransformRcvChan
	_, open := <-b.TransformRcvChan
	if open {
		t.Errorf("Bad client wasn't rejected")
	}
	wg.Done()
}*/

func goodClient(b Portal, expecting int, t *testing.T, wg *sync.WaitGroup) {
	changes := b.BaseVersion() + 1
	seen := 0
	for tform := range b.TransformReadChan() {
		seen++
		if tform.Insert != fmt.Sprintf("%v", changes) {
			t.Errorf("Wrong order of transforms, expected %v, received %v",
				changes, tform.Insert)
		}
		changes++
	}
	if seen != expecting {
		t.Errorf("Good client didn't receive all expected transforms: %v != %v", expecting, seen)
	}
	wg.Done()
}

func TestClients(t *testing.T) {
	errChan := make(chan Error)
	doc, _ := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	config := NewConfig()
	config.FlushPeriod = 5000

	wg := sync.WaitGroup{}

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: *doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
	)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	go func() {
		for err := range errChan {
			t.Errorf("From error channel: %v", err.Err)
		}
	}()

	tform := func(i int) text.OTransform {
		return text.OTransform{
			Position: 0,
			Version:  i,
			Delete:   0,
			Insert:   fmt.Sprintf("%v", i),
		}
	}

	portal, _ := binder.Subscribe("", time.Second)

	if v, err := portal.SendTransform(
		tform(portal.BaseVersion()+1), time.Second); v != 2 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}

	wg.Add(10)
	tformToSend := 50

	for i := 0; i < 10; i++ {
		pt, _ := binder.Subscribe("", time.Second)
		go goodClient(pt, tformToSend, t, &wg)
		//go badClient(binder.Subscribe(""), t, &wg)
	}

	wg.Add(tformToSend)

	for i := 0; i < tformToSend; i++ {
		pt, _ := binder.Subscribe("", time.Second)
		go goodClient(pt, tformToSend-i, t, &wg)
		//go badClient(binder.Subscribe(""), t, &wg)
		if v, err := portal.SendTransform(tform(i+3), time.Second); v != i+3 || err != nil {
			t.Errorf("Send Transform error, expected v: %v, got v: %v, err: %v", i+3, v, err)
		}
	}

	binder.Close()

	wg.Wait()
}

type binderStory struct {
	Content    string            `json:"content" yaml:"content"`
	Transforms []text.OTransform `json:"transforms" yaml:"transforms"`
	TCorrected []text.OTransform `json:"corrected_transforms" yaml:"corrected_transforms"`
	Result     string            `json:"result" yaml:"result"`
}

type binderStoriesContainer struct {
	Stories []binderStory `json:"binder_stories" yaml:"binder_stories"`
}

func goodStoryClient(b Portal, bstory *binderStory, wg *sync.WaitGroup, t *testing.T) {
	tformIndex, lenCorrected := 0, len(bstory.TCorrected)
	go func() {
		for tform := range b.TransformReadChan() {
			if tform.Version != bstory.TCorrected[tformIndex].Version ||
				tform.Insert != bstory.TCorrected[tformIndex].Insert ||
				tform.Delete != bstory.TCorrected[tformIndex].Delete ||
				tform.Position != bstory.TCorrected[tformIndex].Position {
				t.Errorf("Transform not expected, %v != %v", tform, bstory.TCorrected[tformIndex])
			}
			tformIndex++
			if tformIndex == lenCorrected {
				wg.Done()
				return
			}
		}
		t.Errorf("channel was closed before receiving last change")
		wg.Done()
		return
	}()
}

func TestBinderStories(t *testing.T) {
	nClients := 10
	logger, stats := loggerAndStats()

	bytes, err := ioutil.ReadFile("../../test/stories/binder_stories.js")
	if err != nil {
		t.Errorf("Read file error: %v", err)
		return
	}

	var scont binderStoriesContainer
	if err := json.Unmarshal(bytes, &scont); err != nil {
		t.Errorf("Story parse error: %v", err)
		return
	}

	for _, story := range scont.Stories {
		doc, err := store.NewDocument(story.Content)
		if err != nil {
			t.Errorf("error: %v", err)
			continue
		}

		config := NewConfig()
		//config.LogVerbose = true

		errChan := make(chan Error)
		go func() {
			for err := range errChan {
				t.Errorf("From error channel: %v", err.Err)
			}
		}()

		binder, err := New(
			doc.ID,
			&testStore{documents: map[string]store.Document{doc.ID: *doc}},
			config,
			errChan,
			logger,
			stats,
		)
		if err != nil {
			t.Errorf("error: %v", err)
			continue
		}

		wg := sync.WaitGroup{}
		wg.Add(nClients)

		for j := 0; j < nClients; j++ {
			pt, _ := binder.Subscribe("", time.Second)
			goodStoryClient(pt, &story, &wg, t)
		}

		time.Sleep(10 * time.Millisecond)

		bp, _ := binder.Subscribe("", time.Second)
		go func() {
			for range bp.TransformReadChan() {
			}
		}()

		for j := 0; j < len(story.Transforms); j++ {
			if _, err = bp.SendTransform(story.Transforms[j], time.Second); err != nil {
				t.Errorf("Send issue %v", err)
			}
		}

		wg.Wait()

		newClient, _ := binder.Subscribe("", time.Second)
		if got, exp := newClient.Document().Content, story.Result; got != exp {
			t.Errorf("Wrong result, expected: %v, received: %v", exp, got)
		}

		binder.Close()
	}
}
