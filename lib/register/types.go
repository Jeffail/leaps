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

/*
Package register - A collection of types used for registering endpoints across the leaps library.
*/
package register

import "net/http"

/*--------------------------------------------------------------------------------------------------
 */

/*
EndpointRegister - An interface for registering HTTP API endpoints.
*/
type EndpointRegister interface {
	// Register - Register an endpoint handler with a description.
	Register(endpoint, description string, handler http.HandlerFunc)
}

/*
PubPrivEndpointRegister - An interface for registering public or private HTTP API endpoints. The
public endpoints are user facing and the private endpoints are intended for internal administrative
usage.
*/
type PubPrivEndpointRegister interface {
	// RegisterPrivate - Register a public endpoint handler with a description.
	RegisterPublic(endpoint, description string, handler http.HandlerFunc) error

	// RegisterPrivate - Register a private endpoint handler with a description.
	RegisterPrivate(endpoint, description string, handler http.HandlerFunc) error
}

/*--------------------------------------------------------------------------------------------------
 */
