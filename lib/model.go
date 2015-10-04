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

/*--------------------------------------------------------------------------------------------------
 */

/*
ModelConfig - Holds configuration options for a transform model.
*/
type ModelConfig struct {
	MaxDocumentSize    uint64 `json:"max_document_size" yaml:"max_document_size"`
	MaxTransformLength uint64 `json:"max_transform_length" yaml:"max_transform_length"`
}

/*
DefaultModelConfig - Returns a default ModelConfig.
*/
func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		MaxDocumentSize:    50000000, // ~50MB
		MaxTransformLength: 50000,    // ~50KB
	}
}

/*
Model - an interface that represents an internal operation transform model of a particular type.
Initially text is the only supported transform model, however, the plan will eventually be to have
different models for various types of document that should all be supported by our binder.
*/
type Model interface {
	/* PushTransform - Push a single transform to our model, and if successful, return the updated
	 * transform along with the new version of the document.
	 */
	PushTransform(ot OTransform) (OTransform, int, error)

	/* Returns true, if the model have unapplied transforms that can be flushed */
	IsDirty() bool

	/* FlushTransforms - apply all unapplied transforms to content, and delete old applied
	 * in accordance with our retention period. Returns a bool indicating whether any changes
	 * were applied, and an error in case a fatal problem was encountered.
	 */
	FlushTransforms(content *string, secondsRetention int64) (bool, error)

	/* GetVersion - returns the current version of the document.
	 */
	GetVersion() int
}

/*--------------------------------------------------------------------------------------------------
 */
