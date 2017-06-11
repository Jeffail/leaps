{ "client_stories" : [
{
	"name" : "verybasictest",
	"content" : "hello world",
	"result" : "hello crazy dumb internet",
	"epochs" : [
		{
			"send" : [
				{ "position" : 6, "num_delete" : 0, "insert" : "crazy " }
			],
			"receive" : [
				{
					"type" : "correction",
					"body": {
						"correction": {
							"version" : 2
						}
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 12, "num_delete" : 0, "insert" : "dumb ", "version" : 3 },
							{ "position" : 17, "num_delete" : 5, "insert" : "internet", "version" : 4 }
						]
					}
				}
			],
			"result" : "hello crazy dumb internet"
		}
	]
},
{
	"name" : "verybasictest2",
	"content" : "hello world",
	"result" : "hello crazy dumb internet",
	"epochs" : [
		{
			"send" : [],
			"receive" : [
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "dumb ", "version" : 2 },
							{ "position" : 11, "num_delete" : 5, "insert" : "internet", "version" : 3 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "crazy ", "version" : 4 }
						]
					}
				}
			],
			"result" : "hello crazy dumb internet"
		}
	]
},
{
	"name" : "fraggedreqtest",
	"content" : "hello world",
	"result" : "hello stupid internet you fool",
	"epochs" : [
		{
			"send" : [
				{ "position" : 11, "num_delete" : 0, "insert" : " you fool" }
			],
			"receive" : [
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 5, "insert" : "", "version" : 2 },
							{ "position" : 6, "num_delete" : 0, "insert" : "internet", "version" : 3 }
						]
					}
				},
				{
					"type" : "correction",
					"body": {
						"correction": {
							"version" : 4
						}
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "stupid ", "version" : 5 }
						]
					}
				}
			],
			"result" : "hello stupid internet you fool"
		}
	]
},
{
	"name" : "simpleepochstest",
	"content" : "hello world",
	"result" : "hello one two three four five six world",
	"epochs" : [
		{
			"send" : [],
			"receive" : [
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "two ", "version" : 2 },
							{ "position" : 6, "num_delete" : 0, "insert" : "one ", "version" : 3 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 14, "num_delete" : 0, "insert" : "five ", "version" : 4 }
						]
					}
				}
			],
			"result" : "hello one two five world"
		},
		{
			"send" : [],
			"receive" : [
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 14, "num_delete" : 0, "insert" : "three ", "version" : 5 },
							{ "position" : 19, "num_delete" : 0, "insert" : " four", "version" : 6 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 30, "num_delete" : 0, "insert" : "six ", "version" : 7 }
						]
					}
				}
			],
			"result" : "hello one two three four five six world"
		}
	]
},
{
	"name" : "fraggedtest",
	"content" : "hello world",
	"result" : "heyo testing world you poor fool",
	"epochs" : [
		{
			"send" : [
				{ "position" : 6, "num_delete" : 0, "insert" : "testing " }
			],
			"receive" : [
				{
					"type" : "correction",
					"body": {
						"correction": {
							"version" : 6
						}
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 11, "num_delete" : 0, "insert" : " you fool", "version" : 2 },
							{ "position" : 2, "num_delete" : 3, "insert" : "y", "version" : 3 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 13, "num_delete" : 0, "insert" : " poor", "version" : 4 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 3, "num_delete" : 0, "insert" : "o", "version" : 5 }
						]
					}
				}
			],
			"result" : "heyo testing world you poor fool"
		}
	]
},
{
	"name" : "emojitest2",
	"content" : "hello a world",
	"result" : "hello b happy b faces",
	"epochs" : [
		{
			"send" : [
				{ "position" : 6, "num_delete" : 2, "insert" : "happy " }
			],
			"receive" : [
				{
					"type" : "correction",
					"body": {
						"correction": {
							"version" : 6
						}
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 8, "num_delete" : 5, "insert" : "", "version" : 2 },
							{ "position" : 8, "num_delete" : 0, "insert" : "faces", "version" : 3 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "b ", "version" : 4 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 10, "num_delete" : 0, "insert" : "b ", "version" : 5 }
						]
					}
				}
			],
			"result" : "hello b happy b faces"
		}
	]
},
{
	"name" : "emojitest1",
	"content" : "hello 🙁 world",
	"result" : "hello 😀 happy 😀 faces",
	"epochs" : [
		{
			"send" : [
				{ "position" : 6, "num_delete" : 2, "insert" : "happy " }
			],
			"receive" : [
				{
					"type" : "correction",
					"body": {
						"correction": {
							"version" : 6
						}
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 8, "num_delete" : 5, "insert" : "", "version" : 2 },
							{ "position" : 8, "num_delete" : 0, "insert" : "faces", "version" : 3 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 6, "num_delete" : 0, "insert" : "😀 ", "version" : 4 }
						]
					}
				},
				{
					"type" : "transforms",
					"body": {
						"transforms" : [
							{ "position" : 10, "num_delete" : 0, "insert" : "😀 ", "version" : 5 }
						]
					}
				}
			],
			"result" : "hello 😀 happy 😀 faces"
		}
	]
}
] }
