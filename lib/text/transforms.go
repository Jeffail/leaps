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
	"bytes"
)

//------------------------------------------------------------------------------

// OTransform - A representation of a transformation relating to a leaps
// document. This can either be a text addition, a text deletion, or both.
type OTransform struct {
	Position  int    `json:"position"`
	Delete    int    `json:"num_delete"`
	Insert    string `json:"insert"`
	Version   int    `json:"version"`
	TReceived int64  `json:"received,omitempty"`
}

//------------------------------------------------------------------------------

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
FixOutOfDateTransform - When a transform created for a specific version is later
determined to come after one or more other transforms it can be fixed. This fix
translates the transform such that being applied in the correct order will
preserve the original intention.

In order to apply these fixes this function should be called with the target
transform and the actual versioned transform that the target currently
'believes' it is. So, for example, if the transform was written for version 7
and was actually 10 you would call FixOutOfDateTransform in this order:

FixOutOfDateTransform(target, version7)
FixOutOfDateTransform(target, version8)
FixOutOfDateTransform(target, version9)

Once the transform is adjusted through this fix it can be harmlessly dispatched
to all other clients which will end up with the same document as the client that
submitted this transform.

NOTE: These fixes do not regard or alter the versions of either transform.
*/
func FixOutOfDateTransform(sub, pre *OTransform) {
	// Get insertion lengths (codepoints)
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
FixPrematureTransform - Used by clients to fix incoming and outgoing transforms
when local changes have been applied to a document before being routed through
the server.

In order for a client UI to be unblocking it must apply local changes as the
user types them before knowing the correct order of the change. Therefore, it is
possible to apply a local change before receiving incoming transforms that are
meant to be applied beforehand.

As a solution to those situations this function allows a client to alter and
incoming interations such that if they were to be applied to the local document
after our local change they would result in the same document. The outgoing
transform is also modified for sending out to the server.

It is possible that the local change has already been dispatched to the server,
in which case it is the servers responsibility to fix the transform so that
other clients end up at the same result.

NOTE: These fixes do not regard or alter the versions of either transform.
*/
func FixPrematureTransform(unapplied, unsent *OTransform) {
	var before, after *OTransform

	// Order the OTs by position in the document.
	if unapplied.Position <= unsent.Position {
		before = unapplied
		after = unsent
	} else {
		before = unsent
		after = unapplied
	}

	// Get insertion lengths (codepoints)
	bInsert, aInsert := bytes.Runes([]byte(before.Insert)), bytes.Runes([]byte(after.Insert))
	bLength, aLength := len(bInsert), len(aInsert)

	if before.Delete == 0 {
		after.Position += bLength
	} else if (before.Delete + before.Position) <= after.Position {
		after.Position += (bLength - before.Delete)
	} else {
		posGap := after.Position - before.Position
		excess := intMax(0, before.Delete-posGap)

		if excess > after.Delete {
			before.Delete += (aLength - after.Delete)
			before.Insert = before.Insert + after.Insert
		} else {
			before.Delete = posGap
		}

		after.Delete = intMax(0, after.Delete-excess)
		after.Position = before.Position + bLength
	}
}

// MergeTransforms - Takes two transforms (the next to be sent, and the one that
// follows) and attempts to merge them into one transform. This will not be
// possible with some combinations, and the function returns a boolean to
// indicate whether the merge was successful.
func MergeTransforms(first, second *OTransform) bool {
	var overlap, remainder int

	// Get insertion lengths (codepoints)
	fInsert := bytes.Runes([]byte(first.Insert))
	fLength := len(fInsert)

	if first.Position+fLength == second.Position {
		first.Insert = first.Insert + second.Insert
		first.Delete += second.Delete
		return true
	}
	if second.Position == first.Position {
		remainder = intMax(0, second.Delete-fLength)
		first.Delete += remainder
		first.Insert = second.Insert + first.Insert[second.Delete-remainder:]
		return true
	}
	if second.Position > first.Position && second.Position < (first.Position+fLength) {
		overlap = second.Position - first.Position
		remainder = intMax(0, second.Delete-(fLength-overlap))
		first.Delete += remainder
		first.Insert = first.Insert[0:overlap] + second.Insert +
			first.Insert[intMin(fLength, overlap+second.Delete):]
		return true
	}
	return false
}

//------------------------------------------------------------------------------
