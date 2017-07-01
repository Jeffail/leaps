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

package text

import (
	"errors"
	"fmt"
	"time"
)

//------------------------------------------------------------------------------

// Errors for the internal Operational Transform model.
var (
	ErrTransformNegDelete = errors.New("transform contained negative delete")
	ErrTransformOOB       = errors.New("transform position was out of document bounds")
	ErrTransformTooLong   = errors.New("transform insert length exceeded the limit")
	ErrTransformTooOld    = errors.New("transform diff greater than transform archive")
	ErrTransformSkipped   = errors.New("transform version beyond latest")
)

// OTBufferConfig - Holds configuration options for a transform model.
type OTBufferConfig struct {
	MaxDocumentSize    uint64 `json:"max_document_size" yaml:"max_document_size"`
	MaxTransformLength uint64 `json:"max_transform_length" yaml:"max_transform_length"`
}

// NewOTBufferConfig - Returns a default OTBufferConfig.
func NewOTBufferConfig() OTBufferConfig {
	return OTBufferConfig{
		MaxDocumentSize:    52428800, // 50MiB
		MaxTransformLength: 51200,    // 50KiB
	}
}

//------------------------------------------------------------------------------

// OTBuffer - Buffers a growing stack of operational transforms, adjusting any
// out of date transforms as they are added. When the stack is flushed into a
// document any expired transforms are deleted, transforms that are not yet
// expired will be kept as they are used for adjustments.
type OTBuffer struct {
	virtualLen int
	config     OTBufferConfig
	Version    int
	Applied    []OTransform
	Unapplied  []OTransform
}

// NewOTBuffer - Create a buffer of operational transforms for a document set to
// version 1. Takes the initial content of the document in order to perform
// size and bounds checks on incoming transforms.
func NewOTBuffer(content string, config OTBufferConfig) *OTBuffer {
	return &OTBuffer{
		virtualLen: len(content),
		config:     config,
		Version:    1,
		Applied:    []OTransform{},
		Unapplied:  []OTransform{},
	}
}

//------------------------------------------------------------------------------

// PushTransform - Inserts a transform onto the unapplied stack and increments
// the version number of the document. Whilst doing so it fixes the transform in
// relation to earlier transforms it was unaware of, this fixed version gets
// sent back for distributing across other clients.
func (m *OTBuffer) PushTransform(ot OTransform) (OTransform, int, error) {
	// Perform basic checks on size and bounds.
	// NOTE: It is not appropriate to compare this transform to the document
	// length at this stage since the transform might need version adjustment
	// to correct its bounds WRT previous transforms.
	if ot.Position < 0 {
		return OTransform{}, 0, ErrTransformOOB
	}
	if ot.Delete < 0 {
		return OTransform{}, 0, ErrTransformNegDelete
	}
	if uint64(len(ot.Insert)) > m.config.MaxTransformLength {
		return OTransform{}, 0, ErrTransformTooLong
	}

	lenApplied, lenUnapplied := len(m.Applied), len(m.Unapplied)

	diff := (m.Version + 1) - ot.Version

	if diff > lenApplied+lenUnapplied {
		return OTransform{}, 0, ErrTransformTooOld
	}
	if diff < 0 {
		return OTransform{}, 0, ErrTransformSkipped
	}

	for j := lenApplied - (diff - lenUnapplied); j < lenApplied; j++ {
		FixOutOfDateTransform(&ot, &m.Applied[j])
		diff--
	}
	for j := lenUnapplied - diff; j < lenUnapplied; j++ {
		FixOutOfDateTransform(&ot, &m.Unapplied[j])
	}

	// After adjustment check for document size bounds.
	if uint64(len(ot.Insert)-ot.Delete+m.virtualLen) > m.config.MaxDocumentSize {
		return OTransform{}, 0, ErrTransformTooLong
	}
	if (ot.Position + ot.Delete) > m.virtualLen {
		return OTransform{}, 0, ErrTransformOOB
	}

	m.Version++

	ot.Version = m.Version
	ot.TReceived = time.Now().Unix()

	m.Unapplied = append(m.Unapplied, ot)

	m.virtualLen += (len(ot.Insert) - ot.Delete)

	return ot, m.Version, nil
}

// IsDirty - Check if there is any unapplied transforms.
func (m *OTBuffer) IsDirty() bool {
	return len(m.Unapplied) > 0
}

// GetVersion - returns the current version of the document.
func (m *OTBuffer) GetVersion() int {
	return m.Version
}

// FlushTransforms - apply all unapplied transforms and append them to the
// applied stack, then remove old entries from the applied stack. Accepts
// retention as an indicator for how many seconds applied transforms should be
// retained. Returns a bool indicating whether any changes were applied.
func (m *OTBuffer) FlushTransforms(content *string, secondsRetention int64) (bool, error) {
	transforms := m.Unapplied[:]
	m.Unapplied = []OTransform{}

	lenContent := len(*content)

	runeContent := []rune(*content)

	var i, j int
	var err error
	for i = 0; i < len(transforms); i++ {
		lenContent += (len(transforms[i].Insert) - transforms[i].Delete)
		if uint64(lenContent) > m.config.MaxDocumentSize {
			return i > 0, ErrTransformTooLong
		}
		if err = ApplyTransform(&runeContent, &transforms[i]); err != nil {
			break
		}
	}

	*content = string(runeContent)

	upto := time.Now().Unix() - secondsRetention
	for j = 0; j < len(m.Applied); j++ {
		if m.Applied[j].TReceived > upto {
			break
		}
	}

	applied := m.Applied[j:]
	m.Applied = make([]OTransform, len(transforms)+len(applied))

	copy(m.Applied[:], applied)
	copy(m.Applied[len(applied):], transforms)

	return i > 0, err
}

// ApplyTransform - Apply a specific transform to some content.
func ApplyTransform(content *[]rune, ot *OTransform) error {
	if ot.Delete < 0 {
		return ErrTransformNegDelete
	}
	if ot.Position+ot.Delete > len(*content) {
		return fmt.Errorf(
			"transform position (%v) and deletion (%v) surpassed document content length (%v), offender: %v",
			ot.Position, ot.Delete, len(*content), *ot)
	}

	start := (*content)[:ot.Position]
	middle := []rune(ot.Insert)
	end := (*content)[ot.Position+ot.Delete:]

	startLen, middleLen, endLen := len(start), len(middle), len(end)

	(*content) = make([]rune, startLen+middleLen+endLen)
	copy(*content, start)
	copy((*content)[startLen:], middle)
	copy((*content)[startLen+middleLen:], end)

	return nil
}

//------------------------------------------------------------------------------
