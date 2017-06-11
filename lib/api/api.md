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
			"id": "<string, id of document>"
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

TODO
