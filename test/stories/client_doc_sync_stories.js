{ "stories" : [
{
	"name" : "genstory1",
	"content" : "hello 12345 world",
	"result" : "30 54321 01",
	"local_tform": { "position" : 6, "num_delete" : 5, "insert" : "54321" },
	"remote_tforms" : [
		{ "position" : 0, "num_delete" : 5, "insert" : "30" },
		{ "position" : 9, "num_delete" : 5, "insert" : "01" }
	],
	"corrected_local_tform": { "position" : 3, "num_delete" : 5, "insert" : "54321" },
	"corrected_remote_tforms" : [
		{ "position" : 0, "num_delete" : 5, "insert" : "30" },
		{ "position" : 9, "num_delete" : 5, "insert" : "01" }
	]
},
{
	"name" : "genstory2",
	"content" : "hello 12345 world",
	"result" : "30 54321 01",
	"local_tform": { "position" : 0, "num_delete" : 5, "insert" : "30" },
	"remote_tforms" : [
		{ "position" : 6, "num_delete" : 5, "insert" : "54321" },
		{ "position" : 12, "num_delete" : 5, "insert" : "01" }
	],
	"corrected_local_tform": { "position" : 0, "num_delete" : 5, "insert" : "30" },
	"corrected_remote_tforms" : [
		{ "position" : 3, "num_delete" : 5, "insert" : "54321" },
		{ "position" : 9, "num_delete" : 5, "insert" : "01" }
	]
},
{
	"name" : "genstory3",
	"content" : "hello 12345 world",
	"result" : "hello 543210",
	"local_tform": { "position" : 6, "num_delete" : 11, "insert" : "0" },
	"remote_tforms" : [
		{ "position" : 6, "num_delete" : 3, "insert" : "543" },
		{ "position" : 9, "num_delete" : 2, "insert" : "21" }
	],
	"corrected_local_tform": { "position" : 11, "num_delete" : 6, "insert" : "0" },
	"corrected_remote_tforms" : [
		{ "position" : 6, "num_delete" : 0, "insert" : "543" },
		{ "position" : 9, "num_delete" : 0, "insert" : "21" }
	]
},
{
	"name" : "genstory4",
	"content" : "hello 12345 world",
	"result" : "hello 543210 world",
	"local_tform": { "position" : 6, "num_delete" : 0, "insert" : "0" },
	"remote_tforms" : [
		{ "position" : 6, "num_delete" : 3, "insert" : "543" },
		{ "position" : 9, "num_delete" : 2, "insert" : "21" }
	],
	"corrected_local_tform": { "position" : 11, "num_delete" : 0, "insert" : "0" },
	"corrected_remote_tforms" : [
		{ "position" : 6, "num_delete" : 4, "insert" : "5430" },
		{ "position" : 9, "num_delete" : 3, "insert" : "210" }
	]
},
{
	"name" : "genstory5",
	"content" : "hello 12345 world",
	"result" : "hello_w0",
	"local_tform": { "position" : 5, "num_delete" : 12, "insert" : "_" },
	"remote_tforms" : [
		{ "position" : 12, "num_delete" : 2, "insert" : "w0" }
	],
	"corrected_local_tform": { "position" : 5, "num_delete" : 12, "insert" : "_w0" },
	"corrected_remote_tforms" : [
		{ "position" : 6, "num_delete" : 0, "insert" : "w0" }
	]
},
{
	"name" : "genstory6",
	"content" : "hello 12345 world",
	"result" : "hello_ 12345 w0rld",
	"local_tform": { "position" : 5, "num_delete" : 0, "insert" : "_" },
	"remote_tforms" : [
		{ "position" : 12, "num_delete" : 2, "insert" : "w0" }
	],
	"corrected_local_tform": { "position" : 5, "num_delete" : 0, "insert" : "_" },
	"corrected_remote_tforms" : [
		{ "position" : 13, "num_delete" : 2, "insert" : "w0" }
	]
},
{
	"name" : "genstory7",
	"content" : "hello world",
	"result" : "hello stupid internet you fool",
	"local_tform": { "position" : 11, "num_delete" : 0, "insert" : " you fool" },
	"remote_tforms" : [
		{ "position" : 6, "num_delete" : 5, "insert" : "" },
		{ "position" : 6, "num_delete" : 0, "insert" : "internet" },
		{ "position" : 6, "num_delete" : 0, "insert" : "stupid " }
	],
	"corrected_local_tform": { "position" : 21, "num_delete" : 0, "insert" : " you fool" },
	"corrected_remote_tforms" : [
		{ "position" : 6, "num_delete" : 5, "insert" : "" },
		{ "position" : 6, "num_delete" : 0, "insert" : "internet" },
		{ "position" : 6, "num_delete" : 0, "insert" : "stupid " }
	]
}
] }
