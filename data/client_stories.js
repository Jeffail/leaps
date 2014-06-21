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
					"response_type" : "correction",
					"version" : 2
				},
				{
					"response_type" : "transforms",
					"transforms" : [
						{ "position" : 12, "num_delete" : 0, "insert" : "dumb ", "version" : 3 },
						{ "position" : 17, "num_delete" : 5, "insert" : "internet", "version" : 4 }
					]
				}
			],
			"result" : "hello crazy dumb internet"
		}
	]
}
] }
