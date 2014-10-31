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

package lib

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"time"
)

/*--------------------------------------------------------------------------------------------------
 */

/*
OTransform - A representation of a transformation relating to a leap document. This can either be a
text addition, a text deletion, or both.
*/
type OTransform struct {
	Position  int    `json:"position"`
	Delete    int    `json:"num_delete"`
	Insert    string `json:"insert"`
	Version   int    `json:"version"`
	TReceived int64  `json:"received,omitempty"`
}

/*
OModel - A representation of the transform model surrounding a document session. This keeps track
of changes submitted and recently applied in order to distribute those changes to clients.
*/
type OModel struct {
	config    ModelConfig
	Version   int          `json:"version"`
	Applied   []OTransform `json:"applied"`
	Unapplied []OTransform `json:"unapplied"`
}

/*
CreateTextModel - Returns a fresh transform model, with the version set to 1.
*/
func CreateTextModel(config ModelConfig) Model {
	return &OModel{
		config:    config,
		Version:   1,
		Applied:   []OTransform{},
		Unapplied: []OTransform{},
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
PushTransform - Inserts a transform onto the unapplied stack and increments the version number of
the document. Whilst doing so it fixes the transform in relation to earlier transforms it was
unaware of, this fixed version gets sent back for distributing across other clients.
*/
func (m *OModel) PushTransform(otBoxed interface{}) (interface{}, int, error) {
	ot, ok := otBoxed.(OTransform)
	if !ok {
		return nil, 0, errors.New("received unexpected transform, expected OTransform")
	}
	if ot.Delete < 0 {
		return nil, 0, errors.New("transform contained negative delete")
	}
	if uint64(len(ot.Insert)) > m.config.MaxTransformLength {
		return nil, 0, errors.New("transform insert length exceeded the limit")
	}

	lenApplied, lenUnapplied := len(m.Applied), len(m.Unapplied)

	diff := (m.Version + 1) - ot.Version

	if diff > lenApplied+lenUnapplied {
		return nil, 0, errors.New("transform diff greater than transform archive")
	}
	if diff < 0 {
		return nil, 0, fmt.Errorf(
			"transform version %v greater than expected doc version (%v), offender: %v",
			ot.Version, (m.Version + 1), ot)
	}

	for j := lenApplied - (diff - lenUnapplied); j < lenApplied; j++ {
		updateTransform(&ot, &m.Applied[j])
		diff--
	}
	for j := lenUnapplied - diff; j < lenUnapplied; j++ {
		updateTransform(&ot, &m.Unapplied[j])
	}

	m.Version++

	ot.Version = m.Version
	ot.TReceived = time.Now().Unix()

	m.Unapplied = append(m.Unapplied, ot)

	return ot, m.Version, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
GetVersion - returns the current version of the document.
*/
func (m *OModel) GetVersion() int {
	return m.Version
}

/*
FlushTransforms - apply all unapplied transforms and append them to the applied stack, then remove
old entries from the applied stack. Accepts retention as an indicator for how many seconds applied
transforms should be retained. Returns a bool indicating whether any changes were applied.
*/
func (m *OModel) FlushTransforms(contentBoxed *interface{}, secondsRetention int64) (bool, error) {
	content, ok := (*contentBoxed).(string)
	if !ok {
		return false, fmt.Errorf("received unexpected content, expected string, received %v",
			reflect.TypeOf(*contentBoxed))
	}

	transforms := m.Unapplied[:]
	m.Unapplied = []OTransform{}

	lenContent := len(content)

	runeContent := bytes.Runes([]byte(content))

	var i, j int
	var err error
	for i = 0; i < len(transforms); i++ {
		lenContent += (len(transforms[i].Insert) - transforms[i].Delete)
		if uint64(lenContent) > m.config.MaxDocumentSize {
			return i > 0, errors.New("cannot apply transform, document length would exceed limit")
		}
		if err = m.applyTransform(&runeContent, &transforms[i]); err != nil {
			break
		}
	}

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

	*contentBoxed = string(runeContent)

	return i > 0, err
}

func intMin(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func intMax(left, right int) int {
	if left > right {
		return left
	}
	return right
}

/*
updateTransform - When a transform is speculative it potentially has missed transforms that are
already applied. This method retroactively modifies these transforms in relation to the missed
transforms in order to preserve their intention.

The transform 'sub' is the subject of the update, the transform that was constructed without
regard to an earlier transform.

The transform 'pre' is the preceeding transform that should be used to alter 'sub' to preserve
its originally intended change.
*/
func updateTransform(sub *OTransform, pre *OTransform) {
	subInsert, preInsert := bytes.Runes([]byte(sub.Insert)), bytes.Runes([]byte(pre.Insert))
	subLength, preLength := len(subInsert), len(preInsert)

	if pre.Position <= sub.Position {
		if preLength > 0 && pre.Delete == 0 {
			sub.Position += preLength
		} else if pre.Delete > 0 && (pre.Position+pre.Delete) <= sub.Position {
			sub.Position += (preLength - pre.Delete)
		} else if pre.Delete > 0 && (pre.Position+pre.Delete) > sub.Position {
			overhang := intMin(sub.Delete, (pre.Position+pre.Delete)-sub.Position)
			sub.Delete -= overhang
			sub.Position = pre.Position + preLength
		}
	} else if sub.Delete > 0 && (sub.Position+sub.Delete) > pre.Position {
		posGap := pre.Position - sub.Position
		excess := intMax(0, (sub.Delete - posGap))

		if excess > pre.Delete {
			sub.Delete += (preLength - pre.Delete)

			newInsert := make([]rune, subLength+preLength)
			copy(newInsert[:], subInsert)
			copy(newInsert[subLength:], preInsert)

			sub.Insert = string(newInsert)
		} else {
			sub.Delete = posGap
		}
	}
}

/*
applyTransform - Apply a specific transform to some content.
*/
func (m *OModel) applyTransform(content *[]rune, ot *OTransform) error {
	if ot.Delete < 0 {
		return errors.New("transform contained negative deletion")
	}
	if ot.Position+ot.Delete > len(*content) {
		return fmt.Errorf(
			"transform position (%v) and deletion (%v) surpassed document content length (%v), offender: %v",
			ot.Position, ot.Delete, len(*content), *ot)
	}

	start := (*content)[:ot.Position]
	middle := bytes.Runes([]byte(ot.Insert))
	end := (*content)[ot.Position+ot.Delete:]

	startLen, middleLen, endLen := len(start), len(middle), len(end)

	(*content) = make([]rune, startLen+middleLen+endLen)
	copy(*content, start)
	copy((*content)[startLen:], middle)
	copy((*content)[startLen+middleLen:], end)

	return nil
}

/*--------------------------------------------------------------------------------------------------
 */
