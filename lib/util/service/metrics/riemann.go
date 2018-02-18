/*
Copyright (c) 2014 Ashley Jeffs

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

package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/amir/raidman"
)

//--------------------------------------------------------------------------------------------------

func init() {
	constructors["riemann"] = typeSpec{
		constructor: NewRiemann,
		description: `
Benthos can send metrics to Riemann as events, you can set your own tags but it
is recommended that you ensure the 'meter' tag is there to ensure they are dealt
with correctly within Riemann.`,
	}
}

//--------------------------------------------------------------------------------------------------

// RiemannConfig - Configuration fields for a riemann service.
type RiemannConfig struct {
	Server        string   `json:"server" yaml:"server"`
	TTL           float32  `json:"ttl" yaml:"ttl"`
	Tags          []string `json:"tags" yaml:"tags"`
	FlushInterval string   `json:"flush_interval" yaml:"flush_interval"`
	Prefix        string   `json:"prefix" yaml:"prefix"`
}

// NewRiemannConfig - Create a new riemann config with default values.
func NewRiemannConfig() RiemannConfig {
	return RiemannConfig{
		Server:        "",
		TTL:           5,
		Tags:          []string{"service", "meter"},
		FlushInterval: "2s",
		Prefix:        "",
	}
}

//--------------------------------------------------------------------------------------------------

// Riemann - A Riemann client that supports the Type interface.
type Riemann struct {
	sync.Mutex

	config RiemannConfig

	flatMetrics map[string]int64

	Client      *raidman.Client
	eventsCache map[string]*raidman.Event

	flushInterval time.Duration
	quit          chan bool
}

// NewRiemann - Create a new riemann client.
func NewRiemann(config Config) (Type, error) {
	interval, err := time.ParseDuration(config.Riemann.FlushInterval)
	if nil != err {
		return nil, fmt.Errorf("failed to parse flush interval: %v", err)
	}

	client, err := raidman.Dial("tcp", config.Riemann.Server)
	if err != nil {
		return nil, err
	}

	r := &Riemann{
		config:        config.Riemann,
		Client:        client,
		flushInterval: interval,
		eventsCache:   make(map[string]*raidman.Event),
		quit:          make(chan bool),
	}

	go r.loop()

	return r, nil
}

//--------------------------------------------------------------------------------------------------

// Incr - Increment a stat by a value.
func (r *Riemann) Incr(stat string, value int64) error {
	r.Lock()
	defer r.Unlock()

	total, _ := r.flatMetrics[stat]
	total += value

	r.flatMetrics[stat] = total

	service := r.config.Prefix + stat
	r.eventsCache[service] = &raidman.Event{
		Ttl:     r.config.TTL,
		Tags:    r.config.Tags,
		Metric:  total,
		Service: service,
	}
	return nil
}

// Decr - Decrement a stat by a value.
func (r *Riemann) Decr(stat string, value int64) error {
	r.Lock()
	defer r.Unlock()

	total, _ := r.flatMetrics[stat]
	total -= value

	r.flatMetrics[stat] = total

	service := r.config.Prefix + stat
	r.eventsCache[service] = &raidman.Event{
		Ttl:     r.config.TTL,
		Tags:    r.config.Tags,
		Metric:  total,
		Service: service,
	}
	return nil
}

// Timing - Set a stat representing a duration.
func (r *Riemann) Timing(stat string, delta int64) error {
	r.Lock()
	defer r.Unlock()

	service := r.config.Prefix + stat
	r.eventsCache[service] = &raidman.Event{
		Ttl:     r.config.TTL,
		Tags:    r.config.Tags,
		Metric:  delta,
		Service: service,
	}
	return nil
}

// Gauge - Set a stat as a gauge value.
func (r *Riemann) Gauge(stat string, value int64) error {
	r.Lock()
	defer r.Unlock()

	service := r.config.Prefix + stat
	r.eventsCache[service] = &raidman.Event{
		Ttl:     r.config.TTL,
		Tags:    r.config.Tags,
		Metric:  value,
		Service: service,
	}
	return nil
}

// Close - Close the riemann client and stop batch uploading.
func (r *Riemann) Close() error {
	close(r.quit)
	return nil
}

//--------------------------------------------------------------------------------------------------

func (r *Riemann) loop() {
	ticker := time.NewTicker(r.flushInterval)
	for {
		select {
		case <-ticker.C:
			r.flushMetrics()
		case <-r.quit:
			r.Client.Close()
			return
		}
	}
}

func (r *Riemann) flushMetrics() {
	r.Lock()
	defer r.Unlock()

	events := make([]*raidman.Event, len(r.eventsCache))
	i := 0
	for _, event := range r.eventsCache {
		events[i] = event
		i++
	}

	if err := r.Client.SendMulti(events); err == nil {
		r.eventsCache = make(map[string]*raidman.Event)
	} else {
		var newClient *raidman.Client
		newClient, err = raidman.DialWithTimeout("tcp", r.config.Server, r.flushInterval)
		if err == nil {
			r.Client.Close()
			r.Client = newClient
		}
	}
}

//--------------------------------------------------------------------------------------------------
