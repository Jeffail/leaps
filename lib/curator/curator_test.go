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

package curator

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jeffail/leaps/lib/acl"
	"github.com/jeffail/leaps/lib/binder"
	"github.com/jeffail/leaps/lib/store"
	"github.com/jeffail/leaps/lib/text"
	"github.com/jeffail/util/log"
	"github.com/jeffail/util/metrics"
)

func loggerAndStats() (log.Modular, metrics.Aggregator) {
	logConf := log.NewLoggerConfig()
	logConf.LogLevel = "OFF"

	logger := log.NewLogger(os.Stdout, logConf)
	stats := metrics.DudType{}

	return logger, stats
}

func authAndStore(logger log.Modular, stats metrics.Aggregator) (acl.Authenticator, store.Type) {
	return &acl.Anarchy{AllowCreate: true}, store.NewMemory()
}

func TestNewCurator(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	cur, err := New(NewConfig(), log, stats, auth, storage)
	if err != nil {
		t.Errorf("Create curator error: %v", err)
		return
	}

	cur.Close()
}

func TestReadOnlyCurator(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	storage.Create(store.Document{
		ID:      "exists",
		Content: "hello world2",
	})

	curator, err := New(NewConfig(), log, stats, auth, storage)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	doc := store.NewDocument("hello world")
	portal, err := curator.CreateDocument("", "", doc, time.Second)
	doc = portal.Document()
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	if _, err := curator.ReadDocument("test not exist", "", "doesn't exist", time.Second); err == nil {
		t.Error("expected error from non existing document read")
	}

	if _, err := curator.ReadDocument("exists test", "", "exists", time.Second); err != nil {
		t.Errorf("error: %v", err)
		return
	}

	readOnlyPortal, err := curator.ReadDocument("test", "", doc.ID, time.Second)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	if _, err := readOnlyPortal.SendTransform(
		text.OTransform{}, time.Second,
	); err != binder.ErrReadOnlyPortal {
		t.Errorf("read only portal unexpected error: %v", err)
		return
	}

	curator.Close()
}

type dummyAuth struct {
	level acl.AccessLevel
}

func (d *dummyAuth) Authenticate(userID, token, documentID string) acl.AccessLevel {
	return d.level
}

func TestPermissions(t *testing.T) {
	log, stats := loggerAndStats()
	_, storage := authAndStore(log, stats)
	auth := dummyAuth{level: acl.NoAccess}

	cur, err := New(NewConfig(), log, stats, &auth, storage)
	if err != nil {
		t.Errorf("Create curator error: %v", err)
		return
	}

	// No access
	if bNil, err := cur.CreateDocument("", "", store.NewDocument("test"), time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from create on no access")
	}
	if bNil, err := cur.EditDocument("", "", "test", time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from edit on no access")
	}
	if bNil, err := cur.ReadDocument("", "", "test", time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from edit on no access")
	}

	// Read access
	auth.level = acl.ReadAccess
	if bNil, err := cur.CreateDocument("", "", store.NewDocument("test"), time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from create on read access")
	}
	if bNil, err := cur.EditDocument("", "", "test", time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from edit on read access")
	}

	// Edit access
	auth.level = acl.EditAccess
	if bNil, err := cur.CreateDocument("", "", store.NewDocument("test"), time.Second); err == nil || bNil != nil {
		t.Error("Expected rejection from create on edit access")
	}

	cur.Close()
}

func TestGetUsers(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	testCases := map[string][]string{
		"doc1": {
			"user1", "user2", "user3",
		},
		"doc2": {
			"user4", "user5", "user6",
		},
		"doc3": {
			"user1", "user4", "user7",
		},
	}

	for docID := range testCases {
		doc := store.Document{}
		doc.ID = docID
		doc.Content = "hello world"
		if err := storage.Create(doc); err != nil {
			t.Error(err)
		}
	}

	curator, err := New(NewConfig(), log, stats, auth, storage)
	if err != nil {
		t.Error(err)
		return
	}
	defer curator.Close()

	for docID, users := range testCases {
		for _, userID := range users {
			if _, err := curator.EditDocument(userID, "", docID, time.Second); err != nil {
				t.Error(err)
			}
		}
	}

	getUsers, err := curator.GetUsers(time.Second)
	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(getUsers, testCases) {
		t.Errorf("GetUsers not matched, %v != %v", getUsers, testCases)
	}
}

func goodClient(b binder.Portal, expecting int, t *testing.T, wg *sync.WaitGroup) {
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

type dummyBinder struct {
	kickChan   chan string
	closedChan chan struct{}
	id         string
}

func (d *dummyBinder) ID() string {
	return d.id
}

func (d *dummyBinder) GetUsers(timeout time.Duration) ([]string, error) {
	return []string{}, nil
}

func (d *dummyBinder) KickUser(userID string, timeout time.Duration) error {
	d.kickChan <- userID
	return nil
}

func (d *dummyBinder) Subscribe(userID string, timeout time.Duration) (binder.Portal, error) {
	return nil, nil
}

// SubscribeReadOnly - Register a new client as a read only viewer of this binder document.
func (d *dummyBinder) SubscribeReadOnly(userID string, timeout time.Duration) (binder.Portal, error) {
	return nil, nil
}

// Close - Close the binder and shut down all clients, also flushes and cleans up the document.
func (d *dummyBinder) Close() {
	close(d.closedChan)
}

func TestCuratorBinderKicking(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	conf := NewConfig()
	conf.BinderConfig.CloseInactivityPeriodMS = 1

	curator, err := New(conf, log, stats, auth, storage)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	bOne := &dummyBinder{
		id:         "first",
		closedChan: make(chan struct{}),
		kickChan:   make(chan string),
	}

	curator.binderMutex.Lock()
	curator.openBinders = map[string]binder.Type{
		bOne.id: bOne,
	}
	curator.binderMutex.Unlock()

	go func() {
		if err := curator.KickUser("first", "test1", time.Second); err != nil {
			t.Error(err)
		}
	}()

	select {
	case kicked := <-bOne.kickChan:
		if exp, actual := "test1", kicked; exp != actual {
			t.Errorf("Wrong user kicked: %v != %v", exp, actual)
		}
	case <-time.After(time.Second):
		t.Error("User was not kicked")
	}

	if err := curator.KickUser("does not exist", "test1", time.Second); err == nil {
		t.Error("Expected error from invalid binder kicking")
	}

	curator.Close()
}

func TestCuratorBinderClosure(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	conf := NewConfig()
	conf.BinderConfig.CloseInactivityPeriodMS = 1

	curator, err := New(conf, log, stats, auth, storage)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	bOne := &dummyBinder{id: "first", closedChan: make(chan struct{})}
	bTwo := &dummyBinder{id: "second", closedChan: make(chan struct{})}

	curator.binderMutex.Lock()
	curator.openBinders = map[string]binder.Type{
		bOne.id: bOne,
		bTwo.id: bTwo,
	}
	curator.binderMutex.Unlock()

	select {
	case curator.errorChan <- binder.Error{ID: "this doesnt exist, hope we dont panic!", Err: nil}:
	case <-time.After(time.Second):
		t.Error("timed out sending binder error")
	}
	select {
	case curator.errorChan <- binder.Error{ID: bOne.id, Err: nil}:
	case <-time.After(time.Second):
		t.Error("timed out sending binder error")
	}
	select {
	case curator.errorChan <- binder.Error{ID: bTwo.id, Err: errors.New("this was an error")}:
	case <-time.After(time.Second):
		t.Error("timed out sending binder error")
	}

	select {
	case <-bOne.closedChan:
	case <-time.After(time.Second):
		t.Error("binder one was not closed")
	}
	select {
	case <-bTwo.closedChan:
	case <-time.After(time.Second):
		t.Error("binder two was not closed")
	}

	curator.binderMutex.Lock()
	if exp, actual := 0, len(curator.openBinders); exp != actual {
		t.Errorf("Wrong count of openBinders: %v != %v", exp, actual)
	}
	curator.binderMutex.Unlock()

	curator.Close()
}

func TestCuratorClients(t *testing.T) {
	log, stats := loggerAndStats()
	auth, storage := authAndStore(log, stats)

	curator, err := New(NewConfig(), log, stats, auth, storage)
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	doc := store.NewDocument("hello world")
	portal, err := curator.CreateDocument("", "", doc, time.Second)
	doc = portal.Document()
	if err != nil {
		t.Errorf("error: %v", err)
	}

	tform := func(i int) text.OTransform {
		return text.OTransform{
			Position: 0,
			Version:  i,
			Delete:   0,
			Insert:   fmt.Sprintf("%v", i),
		}
	}

	if v, err := portal.SendTransform(
		tform(portal.BaseVersion()+1), time.Second,
	); v != 2 || err != nil {
		t.Errorf("Send Transform error, v: %v, err: %v", v, err)
	}

	wg := sync.WaitGroup{}
	wg.Add(10)

	tformSending := 50

	for i := 0; i < 10; i++ {
		if b, e := curator.EditDocument("test", "", doc.ID, time.Second); e != nil {
			t.Errorf("error: %v", e)
		} else {
			go goodClient(b, tformSending, t, &wg)
		}
		/*if b, e := curator.EditDocument("", doc.ID); e != nil {
			t.Errorf("error: %v", e)
		} else {
			go badClient(b, t, &wg)
		}*/
	}

	wg.Add(25)

	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			if b, e := curator.EditDocument(
				fmt.Sprintf("test%v", i), "", doc.ID, time.Second,
			); e != nil {
				t.Errorf("error: %v", e)
			} else {
				go goodClient(b, tformSending-i, t, &wg)
			}
			/*if b, e := curator.EditDocument("", doc.ID); e != nil {
				t.Errorf("error: %v", e)
			} else {
				go badClient(b, t, &wg)
			}*/
		}
		if v, err := portal.SendTransform(tform(i+3), time.Second); v != i+3 || err != nil {
			t.Errorf("Send Transform error, expected v: %v, got v: %v, err: %v", i+3, v, err)
		}
	}

	closeChan := make(chan bool)

	go func() {
		curator.Close()
		wg.Wait()
		closeChan <- true
	}()

	go func() {
		time.Sleep(1 * time.Second)
		closeChan <- false
	}()

	if closeStatus := <-closeChan; !closeStatus {
		t.Errorf("Timeout occurred waiting for test finish.")
	}
}
