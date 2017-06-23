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

package events

// Metadata may be sent and dispatched from the leaps API in any format, but it
// is useful for clients if we try to maintain a type/body format for easy
// routing on the client side.
//
// The enforced API message structure for a metadata message is as follows:
// {
//   "type": "metadata",
//   "body": {
//     "metadata": <ANYTHING>
//   }
// }
//
// We implement but do not enforce the following format:
// {
//   "type": "metadata",
//   "body": {
//     "metadata": {
//       "type": "<metadata_subtype>",
//       "body": <body_object_for_metadata_subtype>
//     }
//   }
// }

//------------------------------------------------------------------------------

// All explicitly defined outbound/inbound metadata subtypes.
const (
	// UserInfo metadata subtype
	// Server: Send a newly connected client a list of existing clients and
	// their subscriptions, as well the users own username and session id
	UserInfo = "user_info"

	// UserConnected metadata type
	// Server: Send on client connect to leaps service to all other clients
	UserConnect = "user_connect"

	// UserDisconnected metadata type
	// Server: Send on client disconnect to leaps service to all other clients
	UserDisconnect = "user_disconnect"

	// UserSubscribe metadata type
	// Server: Send on client subscribe to document to all other clients
	UserSubscribe = "user_subscribe"

	// UserUnsubscribe metadata type
	// Server: Send on client unsubscribe to document to all other clients
	UserUnsubscribe = "user_unsubscribe"

	// CMDList metadata type
	// Server: Send a newly connected client a list of available static commands
	// that can be run through leaps
	CMDList = "cmd_list"

	// CMD metadata type
	// Client: Submit a command to be run by the leaps service
	CMD = "cmd"

	// CMDOutput metadata type
	// Server: Send the result of a command to all clients connected to the
	// leaps service
	CMDOutput = "cmd_output"
)

//------------------------------------------------------------------------------

// UserSubscriptions contains user identifying information and a list of their
// active subscriptions (document IDs).
type UserSubscriptions struct {
	Username      string   `json:"username"`
	Subscriptions []string `json:"subscriptions"`
}

// CMDData contains the id and the results from a command (if applicable).
type CMDData struct {
	ID     int    `json:"id"`
	Error  string `json:"error"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

//------------------------------------------------------------------------------

// UserInfoMetadataMessage is a metadata body encompassing a map of connected
// session_id's and their document subscriptions.
type UserInfoMetadataMessage struct {
	// Users is a map of session_id to user map objects
	Users map[string]UserSubscriptions `json:"users"`
}

// CMDListMetadataMessage is a metadata body which carries a list of statically
// defined commands available to clients.
type CMDListMetadataMessage struct {
	CMDS []string `json:"cmds"`
}

// CMDMetadataMessage is a metadata body which carries the id and output from
// a command to all clients connected to the leaps service.
type CMDMetadataMessage struct {
	CMDData CMDData `json:"cmd"`
}

// MetadataBody is a message body for typed metadata.
type MetadataBody struct {
	Type string      `json:"type"`
	Body interface{} `json:"body"`
}

//------------------------------------------------------------------------------
