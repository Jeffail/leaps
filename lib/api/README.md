Leaps API
=========

This document outlines the potential Leaps API request/responses in JSON format.
Transport between the client and server is asynchronous, and currently
implemented with websockets.

To begin a connection the client must connect to the endpoint:
`ws://<server>:<port>/leaps/ws?username=<username>`, note that the username is
added to the URL in the query params.

Once the websockets connection is established the server/client communications
are confined to the following JSON format:

```json
{
	"type": "<string>",
	"body": {
		"error": {
			"type": "<string>",
			"message": "<string>"
		},
		"client": {
			"username": "<string>",
			"session_id": "<string>"
		},
		"transform": {
			"insert": "<string>",
			"position": "<int>",
			"num_delete": "<int>"
		},
		"correction": {
			"version": "<int>"
		},
		"metadata": {
			"type": "<string>",
			"body": "<object>"
		},
		"document": {
			"id": "<string>",
			"content": "<string>",
			"version": "<int>"
		}
	}
}
```

## Request/Response Types

### Client Request Types

Clients can send requests of the following types: `subscribe`, `unsubscribe`,
`transform`, `metadata`, `global_metadata`, `ping`.

Which perform the following actions:

#### Subscribe

In order to start editing a document it must be subscribed to. The client makes
a `subscribe` type request which details the document that it intends to edit.

The request looks as follows:

```json
{
	"type": "subscribe",
	"body": {
		"document": {
			"id": "<string, id of document>"
		}
	}
}
```

The service then will respond with either a `subscribe` or an `error` event.

#### Unsubscribe

When a document subscription is active and the client no longer has an interest
in it they can use the `unsubscribe` request, which looks as follows:

```json
{
	"type": "unsubscribe",
	"body": {
		"document": {
			"id": "<string, id of document>"
		}
	}
}
```

The service then will respond with either an `unsubscribe` or an `error` event.

#### Transform

When a subscribed document is edited by the client it must submit a `transform`
request, which looks as follows:

```json
{
	"type": "transform",
	"body": {
		"document": {
			"id": "<string, id of target document>"
		},
		"transform": {
			"insert": "<string, text to insert>",
			"position": "<int, position of change>",
			"num_delete": "<int, number of characters to delete>"
		}
	}
}
```

The service will respond with either a `correction` or an `error` event.

#### Metadata

Sometimes clients need to send their own custom data to other clients. Leaps
will route any `metadata` type of message to other clients subscribed to the
same document, which looks as follows:

```json
{
	"type": "metadata",
	"body": {
		"document": {
			"id": "<string, id of target document>"
		},
		"metadata": {
			"type": "<string, type of metadata>",
			"body": "<object, the metadata itself>"
		}
	}
}
```

The service will not respond to a `metadata` request unless an error occurs.

#### Global Metadata

Clients may also send metadata to _all_ users connected to the leaps service,
regardless of subscription. This is done with the `global_metadata` event,
which is similar in all other ways to the `metadata` event, and looks as
follows:

```json
{
	"type": "global_metadata",
	"body": {
		"metadata": {
			"type": "<string, type of metadata>",
			"body": "<object, the metadata itself>"
		}
	}
}
```

The service will not respond to a `global_metadata` request unless an error
occurs.

#### Ping

Send a `ping` event to get back a `pong` event, there is no body to the request
so it looks like this:

```json
{
	"type": "ping",
	"body": {}
}
```

### Server Response Types

Servers will send responses of the following types: `subscribe`, `unsubscribe`,
`correction`, `transforms`, `metadata`, `global_metadata`, `pong`.

Which perform the following actions:

#### Subscribe

When a client makes a `subscribe` request, and the request is successful, the
server will also respond with a `subscribe` typed response.

The response looks as follows:

```json
{
	"type": "subscribe",
	"body": {
		"document": {
			"id": "<string, id of document>",
			"content": "<string, the current content of the document>",
			"version": "<int, the current version of the document>"
		}
	}
}
```

#### Unsubscribe

When a client makes an `unsubscribe` request, and the request is successful, the
server will also respond with an `unsubscribe` typed response.

The response looks as follows:

```json
{
	"type": "unsubscribe",
	"body": {
		"document": {
			"id": "<string, id of document>"
		}
	}
}
```

#### Correction

When a client submits a transform it is speculative in that the version of the
transform is only what the client _expects_ it to be. The server then processes
the transform, corrects it, and then responds with a `correction` typed response
of the following format:

```json
{
	"type": "correction",
	"body": {
		"document": {
			"id": "<string, id of document>"
		},
		"correction": {
			"version": "<int, the actual version of the last submitted transform>"
		}
	}
}
```

#### Transforms

A document transform submitted by a client will be broadcast to all other
subscribed clients. Those clients receive a `transforms` typed message, as this
message may potentially contain multiple transforms.

The response looks as follows:

```json
{
	"type": "transforms",
	"body": {
		"document": {
			"id": "<string, id of document>"
		},
		"transforms": [
			{
				"insert": "<string, text to insert>",
				"position": "<int, position of change>",
				"num_delete": "<int, number of characters to delete>"
			}
		]
	}
}
```

The service will respond with either a `correction` or an `error` event.

#### Metadata

Metadata submitted from subscribed clients are broadcast to all other subscribed
clients in the same format with additional client identifying information:

```json
{
	"type": "metadata",
	"body": {
		"client": {
			"username": "<string, username of the source client>",
			"session_id": "<string, unique uuid of the source client>"
		},
		"document": {
			"id": "<string, id of document>"
		},
		"metadata": {
			"type": "<string, type of metadata>",
			"body": "<object, the metadata itself>"
		}
	}
}
```

#### Global Metadata

Global metadata submitted from clients are broadcast to all other connected
clients in the same format with additional client identifying information:

```json
{
	"type": "global_metadata",
	"body": {
		"client": {
			"username": "<string, username of the source client>",
			"session_id": "<string, unique uuid of the source client>"
		},
		"metadata": {
			"type": "<string, type of metadata>",
			"body": "<object, the metadata itself>"
		}
	}
}
```

There are a number of `global_metadata` events that the server sends
automatically during a connection, such as `user_info`. To read about these
events, as well as any established client events, you can read the metadata spec
[here][0].

#### Pong

Sent back after receiving a `ping` request.

```json
{
	"type": "pong",
	"body": {}
}
```

[0]: lib/api/metadata.md
