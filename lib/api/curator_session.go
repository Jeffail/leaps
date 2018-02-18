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
	"fmt"
	"sync"
	"time"

	"github.com/Jeffail/leaps/lib/api/events"
	"github.com/Jeffail/leaps/lib/binder"
	"github.com/Jeffail/leaps/lib/curator"
	"github.com/Jeffail/leaps/lib/text"
	"github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
)

//------------------------------------------------------------------------------

// CuratorSession - An API gateway between a client and a leaps curator. This
// gateway is responsible for tracking a client session as it attempts to
// create, connect to and edit documents.
type CuratorSession struct {
	emitter Emitter
	cur     curator.Type

	timeout time.Duration
	logger  log.Modular
	stats   metrics.Type

	username  string
	uuid      string
	portals   map[string]binder.Portal
	portalMut sync.Mutex
}

// NewCuratorSession - Creates a curator gateway API for a user IO session that
// binds to events from the provided network IO emitter. Currently the API is
// limited to single document access, this could, however, be changed later.
func NewCuratorSession(
	username, uuid string,
	emitter Emitter,
	cur curator.Type,
	timeout time.Duration,
	logger log.Modular,
	stats metrics.Type,
) *CuratorSession {
	s := &CuratorSession{
		username: username,
		uuid:     uuid,
		emitter:  emitter,
		portals:  map[string]binder.Portal{},
		cur:      cur,
		timeout:  timeout,
		logger:   logger.NewModule(":api:session"),
		stats:    stats,
	}
	emitter.OnReceive(events.Subscribe, s.subscribe)
	emitter.OnReceive(events.Unsubscribe, s.unsubscribe)
	emitter.OnReceive(events.Transform, s.transform)
	emitter.OnReceive(events.Metadata, s.metadata)
	emitter.OnReceive(events.Ping, s.ping)

	emitter.OnClose(func() {
		s.portalMut.Lock()
		// Clean up any existing subscriptions when the socket is closed.
		for _, p := range s.portals {
			p.Exit(s.timeout)
		}
		s.portalMut.Unlock()
	})
	return s
}

//------------------------------------------------------------------------------

// API Handlers

// Subscribe to a document by providing an ID.
func (s *CuratorSession) subscribe(body []byte) events.TypedError {
	var req events.SubscriptionMessage
	if err := json.Unmarshal(body, &req); err != nil {
		s.stats.Incr("api.session.subscribe.error.json", 1)
		s.logger.Warnf("Subscribe parse error: %v\n", err)
		return events.NewAPIError(events.ErrBadJSON, err.Error())
	}

	s.portalMut.Lock()
	defer s.portalMut.Unlock()

	// The API is currently limited to one subscription per connection.
	if _, exists := s.portals[req.Document.ID]; exists {
		s.stats.Incr("api.session.subscribe.error.already_subscribed", 1)
		return events.NewAPIError(
			events.ErrExistingSub,
			fmt.Sprintf("This session is already subscribed to document %v", req.Document.ID),
		)
	}

	portal, err := s.cur.EditDocument(
		events.Client{Username: s.username, SessionID: s.uuid}, "", req.Document.ID, s.timeout,
	)
	if err != nil {
		s.stats.Incr("api.session.subscribe.error.curator", 1)
		s.logger.Warnf("Subscribe edit error: %v\n", err)
		return events.NewAPIError(events.ErrSubscribe, err.Error())
	}
	s.emitter.Send(events.Subscribe, events.SubscriptionMessage{
		Document: events.DocumentFull{
			ID:      portal.Document().ID,
			Content: portal.Document().Content,
			Version: portal.BaseVersion(),
		},
	})
	s.stats.Incr("api.session.subscribe.success", 1)
	s.stats.Incr("api.session.subscribed", 1)
	s.portals[req.Document.ID] = portal
	portal.ReleaseDocument()

	go func() {
		open := true
		for open {
			var m binder.ClientMetadata
			var t text.OTransform
			select {
			case m, open = <-portal.MetadataReadChan():
				if open {
					s.emitter.Send(events.Metadata, events.MetadataMessage{
						Document: events.DocumentStripped{
							ID: portal.Document().ID,
						},
						Client:   m.Client,
						Metadata: m.Metadata,
					})
				}
			case t, open = <-portal.TransformReadChan():
				if open {
					s.emitter.Send(events.Transforms, events.TransformsMessage{
						Document: events.DocumentStripped{
							ID: portal.Document().ID,
						},
						Transforms: []text.OTransform{t},
					})
				}
			}
		}
		s.portalMut.Lock()
		delete(s.portals, req.Document.ID)
		s.portalMut.Unlock()

		s.stats.Decr("api.session.subscribed", 1)
		s.emitter.Send(events.Unsubscribe, events.UnsubscriptionMessage{
			Document: events.DocumentStripped{
				ID: req.Document.ID,
			},
		})
	}()
	return nil
}

// Unsubscribe from currently subscribed document
func (s *CuratorSession) unsubscribe(body []byte) events.TypedError {
	var req events.UnsubscriptionMessage
	if err := json.Unmarshal(body, &req); err != nil {
		s.stats.Incr("api.session.unsubscribe.error.json", 1)
		s.logger.Warnf("Unsubscribe parse error: %v\n", err)
		return events.NewAPIError(events.ErrBadJSON, err.Error())
	}

	s.portalMut.Lock()
	defer s.portalMut.Unlock()

	portal, exists := s.portals[req.Document.ID]
	if !exists {
		s.stats.Incr("api.session.unsubscribe.error.not_subscribed", 1)
		return events.NewAPIError(
			events.ErrNoSub,
			fmt.Sprintf("This session is not yet subscribed to document %v", req.Document.ID),
		)
	}
	portal.Exit(s.timeout)
	s.stats.Incr("api.session.unsubscribe.success", 1)
	return nil
}

// Submit a transform to the currently subscribed document
func (s *CuratorSession) transform(body []byte) events.TypedError {
	var req events.TransformMessage
	if err := json.Unmarshal(body, &req); err != nil {
		s.stats.Incr("api.session.transform.error.json", 1)
		s.logger.Warnf("Transform parse error: %v\n", err)
		return events.NewAPIError(events.ErrBadJSON, err.Error())
	}

	s.portalMut.Lock()
	defer s.portalMut.Unlock()

	portal, exists := s.portals[req.Document.ID]
	if !exists {
		s.stats.Incr("api.session.transform.error.not_subscribed", 1)
		return events.NewAPIError(
			events.ErrNoSub,
			fmt.Sprintf("This session is not yet subscribed to document %v", req.Document.ID),
		)
	}

	v, err := portal.SendTransform(req.Transform, s.timeout)
	if err != nil {
		s.stats.Incr("api.session.transform.error.send", 1)
		s.logger.Warnf("Transform send error: %v\n", err)
		return events.NewAPIError(events.ErrTransform, err.Error())
	}
	s.stats.Incr("api.session.transform.success", 1)
	s.emitter.Send(events.Correction, events.CorrectionMessage{
		Document: events.DocumentStripped{
			ID: portal.Document().ID,
		},
		Correction: events.TformCorrection{
			Version: v,
		},
	})
	return nil
}

// Broadcast metadata to all other clients of the currently subscribed document
func (s *CuratorSession) metadata(body []byte) events.TypedError {
	var req events.MetadataMessage
	if err := json.Unmarshal(body, &req); err != nil {
		s.stats.Incr("api.session.metadata.error.json", 1)
		s.logger.Warnf("Metadata parse error: %v\n", err)
		return events.NewAPIError(events.ErrBadJSON, err.Error())
	}

	s.portalMut.Lock()
	defer s.portalMut.Unlock()

	portal, exists := s.portals[req.Document.ID]
	if !exists {
		s.stats.Incr("api.session.transform.error.not_subscribed", 1)
		return events.NewAPIError(
			events.ErrNoSub,
			fmt.Sprintf("This session is not yet subscribed to document %v", req.Document.ID),
		)
	}
	s.stats.Incr("api.session.metadata.success", 1)
	portal.SendMetadata(req.Metadata)
	return nil
}

// Ping for a pong
func (s *CuratorSession) ping(body []byte) events.TypedError {
	s.emitter.Send(events.Pong, nil)
	return nil
}

//------------------------------------------------------------------------------
