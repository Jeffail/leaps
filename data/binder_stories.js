{ "binder_stories" : [
{
	"content" : "hello world",
	"result" : "helwhenwat world",
	"transforms" : [
		[{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 }],
		[{ "position" : 3, "num_delete" : 2, "insert" : "when", "version" : 3 }]
	],
	"corrected_transforms" : [
		[{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 }],
		[{ "position" : 3, "num_delete" : 2, "insert" : "when", "version" : 3 }]
	]
}
,{
	"content" : "hello world",
	"result" : "hellwhenwatworld",
	"transforms" : [
		[{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 }],
		[{ "position" : 4, "num_delete" : 2, "insert" : "when", "version" : 2 }]
	],
	"corrected_transforms" : [
		[{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 }],
		[{ "position" : 4, "num_delete" : 5, "insert" : "whenwat", "version" : 3 }]
	]
}
] }
