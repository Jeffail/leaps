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
	"reflect"
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

func TestFailedInitialFlush(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	storage := testStore{documents: nil}

	_, err := New("KILL_ME", &storage, NewConfig(), errChan, logger, stats, nil)
	if err == nil {
		t.Error("Expected error from failed initial flush")
	}
}

type dumbAuditor struct {
	transforms []text.OTransform
}

func (d *dumbAuditor) OnTransform(t text.OTransform) error {
	d.transforms = append(d.transforms, t)
	return nil
}

func TestAuditor(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc := store.NewDocument("hello world")

	storage := testStore{documents: map[string]store.Document{
		"KILL_ME": doc,
	}}

	auditor := &dumbAuditor{}

	binder, err := New("KILL_ME", &storage, NewConfig(), errChan, logger, stats, auditor)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	defer binder.Close()

	if exp, actual := "KILL_ME", binder.ID(); exp != actual {
		t.Errorf("Wrong result from ID call: %v != %v", exp, actual)
	}

	testClient, err := binder.Subscribe("", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 2}, time.Second)

	if exp, actual := 1, len(auditor.transforms); exp != actual {
		t.Errorf("Wrong count of audits: %v != %v", exp, actual)
		return
	}
	if exp, actual := "hello", auditor.transforms[0].Insert; exp != actual {
		t.Errorf("Wrong value within audits: %v != %v", exp, actual)
	}
	if exp, actual := 0, auditor.transforms[0].Position; exp != actual {
		t.Errorf("Wrong value within audits: %v != %v", exp, actual)
	}
	if exp, actual := 2, auditor.transforms[0].Version; exp != actual {
		t.Errorf("Wrong value within audits: %v != %v", exp, actual)
	}
}

func TestGracefullShutdown(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc := store.NewDocument("hello world")

	storage := testStore{documents: map[string]store.Document{
		"KILL_ME": doc,
	}}

	binder, err := New("KILL_ME", &storage, NewConfig(), errChan, logger, stats, nil)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	if exp, actual := "KILL_ME", binder.ID(); exp != actual {
		t.Errorf("Wrong result from ID call: %v != %v", exp, actual)
	}

	testClient, err := binder.Subscribe("", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	delete(storage.documents, "KILL_ME")
	testClient.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 2}, time.Second)

	if bErr := <-errChan; bErr.Err == nil {
		t.Error("Expected an error from errchan")
	}
	binder.Close()
}

func TestClientExitAndShutdown(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc := store.NewDocument("hello world")

	storage := testStore{documents: map[string]store.Document{
		"KILL_ME": doc,
	}}

	conf := NewConfig()
	conf.ClientKickPeriodMS = 1        // Basically do not block at all on clients
	conf.CloseInactivityPeriodMS = 500 // 1 second of inactivity before we close

	binder, err := New("KILL_ME", &storage, conf, errChan, logger, stats, nil)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}

	testClient1, err := binder.Subscribe("1", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	testClient2, err := binder.Subscribe("2", time.Second)
	if err != nil {
		t.Error(err)
		return
	}
	testClient1.SendTransform(text.OTransform{Position: 0, Insert: "hello", Version: 2}, time.Second)
	testClient1.SendTransform(text.OTransform{Position: 5, Insert: " world", Version: 3}, time.Second)

	// Block testClient2 transform chan so that it gets kicked.
	// We read the message channel instead, waiting for it to be closed.
	select {
	case _, open := <-testClient2.MetadataReadChan():
		if open {
			t.Error("Received unexpected update to lazy client")
		}
	case <-time.After(time.Second * 5):
		t.Error("Lazy client was not kicked")
	}

	testClient1.Exit(time.Second)
	// This is the last client, the binder should shut down now.

	select {
	case bErr := <-errChan:
		if bErr.Err != nil {
			t.Errorf("Unexpected error while waiting for graceful shutdown request: %v", bErr.Err)
		}
	case <-time.After(time.Second * 5):
		t.Error("Empty binder was not closed")
	}
	binder.Close()
}

func TestKickLockedUsers(t *testing.T) {
	errChan := make(chan Error, 10)

	logger, stats := loggerAndStats()
	doc := store.NewDocument("hello world")

	store := testStore{documents: map[string]store.Document{
		"KILL_ME": doc,
	}}

	conf := NewConfig()
	conf.ClientKickPeriodMS = 1

	binder, err := New("KILL_ME", &store, conf, errChan, logger, stats, nil)
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
	doc := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
		nil,
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

	if pUserID := portal1.ClientMetadata(); !reflect.DeepEqual(userID1, pUserID) {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID1, pUserID)
	}
	if pUserID := portal2.ClientMetadata(); !reflect.DeepEqual(userID2, pUserID) {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID2, pUserID)
	}

	for i := 0; i < 100; i++ {
		portal1.SendMetadata(i)

		message := <-portal2.MetadataReadChan()
		if !reflect.DeepEqual(message.Client, userID1) {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message.Client, userID1,
			)
		}
		if !reflect.DeepEqual(message.Metadata, i) {
			t.Errorf(
				"Received incorrect metadata content: %v != %v",
				message.Metadata, i,
			)
		}

		portal2.SendMetadata(i)

		message2 := <-portal1.MetadataReadChan()
		if !reflect.DeepEqual(message2.Client, userID2) {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message2.Client, userID2,
			)
		}
		if !reflect.DeepEqual(message2.Metadata, i) {
			t.Errorf(
				"Received incorrect metadata content: %v != %v",
				message2.Metadata, i,
			)
		}
	}
}

func TestUpdatesSameUserID(t *testing.T) {
	errChan := make(chan Error)
	doc := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
		nil,
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

	if pUserID := portal1.ClientMetadata(); !reflect.DeepEqual(userID, pUserID) {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID, pUserID)
	}
	if pUserID := portal2.ClientMetadata(); !reflect.DeepEqual(userID, pUserID) {
		t.Errorf("Binder portal wrong user ID: %v != %v", userID, pUserID)
	}

	for i := 0; i < 100; i++ {
		portal1.SendMetadata(nil)

		message := <-portal2.MetadataReadChan()
		if !reflect.DeepEqual(message.Client, portal1.ClientMetadata()) {
			t.Errorf(
				"Received incorrect user ID: %v != %v",
				message.Client, portal1.ClientMetadata(),
			)
		}

		portal2.SendMetadata(nil)

		message2 := <-portal1.MetadataReadChan()
		if !reflect.DeepEqual(message2.Client, portal2.ClientMetadata()) {
			t.Errorf(
				"Received incorrect token: %v != %v",
				message2.Client, portal2.ClientMetadata(),
			)
		}
	}
}

func TestNew(t *testing.T) {
	errChan := make(chan Error)
	doc := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
		nil,
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
	doc := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
		nil,
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
	doc := store.NewDocument("hello world")
	logger, stats := loggerAndStats()

	config := NewConfig()
	config.FlushPeriodMS = 5000

	wg := sync.WaitGroup{}

	binder, err := New(
		doc.ID,
		&testStore{documents: map[string]store.Document{doc.ID: doc}},
		NewConfig(),
		errChan,
		logger,
		stats,
		nil,
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
		doc := store.NewDocument(story.Content)

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
			&testStore{documents: map[string]store.Document{doc.ID: doc}},
			config,
			errChan,
			logger,
			stats,
			nil,
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
