{ "binder_stories" : [
{
	"name" : "story1",
	"content" : "hello world",
	"result" : "helwhenwat world",
	"transforms" : [
		{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 },
		{ "position" : 3, "num_delete" : 2, "insert" : "when", "version" : 3 }
	],
	"corrected_transforms" : [
		{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 },
		{ "position" : 3, "num_delete" : 2, "insert" : "when", "version" : 3 }
	]
}
,{
	"name" : "story2",
	"content" : "hello world",
	"result" : "hellwhenwatworld",
	"transforms" : [
		{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 },
		{ "position" : 4, "num_delete" : 2, "insert" : "when", "version" : 2 }
	],
	"corrected_transforms" : [
		{ "position" : 5, "num_delete" : 0, "insert" : "wat", "version" : 2 },
		{ "position" : 4, "num_delete" : 5, "insert" : "whenwat", "version" : 3 }
	]
}
,{
	"name" : "story3",
	"content" : "hello world, it was fun.",
	"result" : "goodbye cruel world, it wasn't fun.",
	"transforms" : [
		{ "position" : 0, "num_delete" : 5, "insert" : "good", "version" : 2 },
		{ "position" : 4, "num_delete" : 0, "insert" : "bye", "version" : 3 },
		{ "position" : 5, "num_delete" : 0, "insert" : " cruel", "version" : 2 },
		{ "position" : 19, "num_delete" : 0, "insert" : "n't", "version" : 2 }
	],
	"corrected_transforms" : [
		{ "position" : 0, "num_delete" : 5, "insert" : "good", "version" : 2 },
		{ "position" : 4, "num_delete" : 0, "insert" : "bye", "version" : 3 },
		{ "position" : 7, "num_delete" : 0, "insert" : " cruel", "version" : 4 },
		{ "position" : 27, "num_delete" : 0, "insert" : "n't", "version" : 5 }
	]
}
] }
