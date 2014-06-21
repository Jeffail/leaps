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

package leaplib

import (
	"errors"
	"fmt"
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
	DocID     string        `json:"docid"`
	Version   int           `json:"version"`
	Applied   []*OTransform `json:"applied"`
	Unapplied []*OTransform `json:"unapplied"`
}

/*
CreateModel - Returns a fresh transform model, with the version set to 1.
*/
func CreateModel(id string) *OModel {
	return &OModel{
		Version:   1,
		DocID:     id,
		Applied:   []*OTransform{},
		Unapplied: []*OTransform{},
	}
}

/*--------------------------------------------------------------------------------------------------
 */

/*
PushTransform - Inserts a transform onto the unapplied stack and increments the version number of
the document. Whilst doing so it fixes the transform in relation to earlier transforms it was
unaware of, this fixed version gets sent back for distributing across other clients.
*/
func (m *OModel) PushTransform(ot OTransform) (*OTransform, error) {
	if ot.Delete < 0 {
		return nil, errors.New("transform contained negative delete")
	}

	lenApplied, lenUnapplied := len(m.Applied), len(m.Unapplied)

	diff := (m.Version + 1) - ot.Version

	if diff > lenApplied+lenUnapplied {
		return nil, errors.New("transform diff greater than transform archive")
	}
	if diff < 0 {
		return nil, fmt.Errorf(
			"transform version %v greater than expected doc version (%v)", ot.Version, (m.Version + 1))
	}

	for j := lenApplied - (diff - lenUnapplied); j < lenApplied; j++ {
		updateTransform(&ot, m.Applied[j])
		diff--
	}
	for j := lenUnapplied - diff; j < lenUnapplied; j++ {
		updateTransform(&ot, m.Unapplied[j])
	}

	m.Version++

	ot.Version = m.Version
	ot.TReceived = time.Now().Unix()

	m.Unapplied = append(m.Unapplied, &ot)

	return &ot, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
PushTransforms - Inserts a slice of transforms onto the unapplied stack and increments the version
number of the document for each new item. Whilst doing so it fixes the transforms in relation to
earlier transforms it was unaware of, the fixed version of the base transform (the first) gets sent
back for distributing across other clients.

This method is important for systems where clients can submit multiple transforms simultaneously, as
each transform in the array will be aware of its immediate predecessors, but not the server held
transforms before it.
*/
func (m *OModel) PushTransforms(ots []*OTransform) ([]*OTransform, error) {
	copyOTs := make([]*OTransform, len(ots))
	for i, ot := range ots {
		tmp := *ot
		copyOTs[i] = &tmp
	}

	if len(ots) <= 0 {
		return copyOTs, errors.New("submitted empty slice of transforms")
	}

	baseVersion := copyOTs[0].Version

	lenApplied, lenUnapplied := len(m.Applied), len(m.Unapplied)

	diff := (m.Version + 1) - baseVersion

	if diff > lenApplied+lenUnapplied {
		return copyOTs, errors.New("transform diff greater than transform archive")
	}
	if diff < 0 {
		return copyOTs, fmt.Errorf(
			"transform version %v greater than expected doc version (%v)", baseVersion, (m.Version + 1))
	}

	for _, ot := range copyOTs {
		if ot.Delete < 0 {
			return copyOTs, errors.New("transform contained negative delete")
		}

		for j := lenApplied - (diff - lenUnapplied); j < lenApplied; j++ {
			updateTransform(ot, m.Applied[j])
			diff--
		}
		for j := lenUnapplied - diff; j < lenUnapplied; j++ {
			updateTransform(ot, m.Unapplied[j])
		}
	}

	for _, ot := range copyOTs {
		m.Version++

		ot.Version = m.Version
		ot.TReceived = time.Now().Unix()

		m.Unapplied = append(m.Unapplied, ot)
	}

	return copyOTs, nil
}

/*
GetTransforms - Returns transforms from a document starting from a specific version number. Along
with the current version number of the document. Also returns an error if the target version is so
out of date that the client is going to suffer problems. Take any error here as an indicator that
you should resync with the document.
*/
func (m *OModel) GetTransforms(version int) ([]*OTransform, int, error) {

	diff := m.Version - version
	if diff < 0 {
		return nil, m.Version, errors.New("your target version is greater than the actual version")
	}
	if diff == 0 {
		return []*OTransform{}, m.Version, nil
	}

	numApplied, numUnapplied := len(m.Applied), len(m.Unapplied)
	if diff > (numApplied + numUnapplied) {
		msg := fmt.Sprintf("your target version is out of date (diff: %v)", diff)
		return nil, m.Version, errors.New(msg)
	}

	transforms := make([]*OTransform, diff)

	takingApplied, tookApplied := (diff - numUnapplied), 0

	if takingApplied > 0 {
		for i := 0; i < takingApplied; i++ {
			transforms[i] = &OTransform{}
			*transforms[i] = *m.Applied[(numApplied-takingApplied)+i]
		}
		diff -= takingApplied
		tookApplied = takingApplied
	}

	skipUnapplied := numUnapplied - diff
	for i := 0; i < diff; i++ {
		transforms[i+tookApplied] = &OTransform{}
		*transforms[i+tookApplied] = *m.Unapplied[skipUnapplied+i]
	}

	return transforms, m.Version, nil
}

/*--------------------------------------------------------------------------------------------------
 */

/*
FlushTransforms - apply all unapplied transforms and append them to the applied stack, then remove
old entries from the applied stack. Accepts retention as an indicator for how long applied
transforms should be retained.
*/
func (m *OModel) FlushTransforms(content *string, retention time.Duration) error {
	transforms := m.Unapplied[:]
	m.Unapplied = []*OTransform{}

	byteContent := []byte(*content)

	var i int
	for i = 0; i < len(transforms); i++ {
		if err := m.applyTransform(&byteContent, transforms[i]); err != nil {
			return err
		}
	}

	upto := time.Now().Unix() - int64(retention)
	for i = 0; i < len(m.Applied); i++ {
		if m.Applied[i].TReceived > upto {
			break
		} else {
			m.Applied[i] = nil
		}
	}

	m.Applied = append(m.Applied[i:], transforms...)
	(*content) = string(byteContent)

	return nil
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
	subLength, preLength := len(sub.Insert), len(pre.Insert)
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

			newInsert := make([]byte, subLength+preLength)
			copy(newInsert[:], []byte(sub.Insert))
			copy(newInsert[subLength:], []byte(pre.Insert))

			sub.Insert = string(newInsert)
		} else {
			sub.Delete = posGap
		}
	}
}

/*
applyTransform - Apply a specific transform to some content.
*/
func (m *OModel) applyTransform(content *[]byte, ot *OTransform) error {
	if ot.Delete < 0 {
		return errors.New("transform contained negative deletion")
	}
	if ot.Position+ot.Delete > len(*content) {
		return fmt.Errorf("transform position (%v) and deletion (%v) surpassed document content length (%v)",
			ot.Position, ot.Delete, len(*content))
	}

	start := (*content)[:ot.Position]
	middle := []byte(ot.Insert)
	end := (*content)[ot.Position+ot.Delete:]

	startLen, middleLen, endLen := len(start), len(middle), len(end)

	(*content) = make([]byte, startLen+middleLen+endLen)
	copy(*content, start)
	copy((*content)[startLen:], middle)
	copy((*content)[startLen+middleLen:], end)

	return nil
}

/*--------------------------------------------------------------------------------------------------
 */
