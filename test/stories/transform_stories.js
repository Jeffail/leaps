{ "stories" : [
{
	"name" : "genstory1",
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
},
{
	"name" : "genstory2",
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
},
{
	"name" : "genstory3",
	"content" : "hello world",
	"result" : "hekopjippyp",
	"transforms" : [
		{ "position" : 3, "num_delete" : 0, "insert" : "jip", "version" : 2 },
		{ "position" : 2, "num_delete" : 6, "insert" : "kop", "version" : 2 },
		{ "position" : 4, "num_delete" : 7, "insert" : "pyp", "version" : 2 }
	]
},
{
	"name" : "genstory4",
	"content" : "hello world",
	"result" : "hekopjiprld",
	"transforms" : [
		{ "position" : 3, "num_delete" : 4, "insert" : "jip", "version" : 2 },
		{ "position" : 2, "num_delete" : 6, "insert" : "kop", "version" : 2 }
	]
},
{
	"name" : "genstory5",
	"content" : "hello world",
	"result" : "hekopjiprld",
	"transforms" : [
		{ "position" : 2, "num_delete" : 6, "insert" : "kop", "version" : 2 },
		{ "position" : 3, "num_delete" : 4, "insert" : "jip", "version" : 2 }
	]
},
{
	"name" : "genstory6",
	"content" : "hello world",
	"result" : "hekopjiprld",
	"transforms" : [
		{ "position" : 2, "num_delete" : 6, "insert" : "kop", "version" : 2 },
		{ "position" : 2, "num_delete" : 2, "insert" : "jip", "version" : 2 }
	]
},
{
	"name" : "genstory7",
	"content" : "hello world hello world",
	"result" : "hkoptesting world hello world",
	"transforms" : [
		{ "position" : 2, "num_delete" : 2, "insert" : "testing", "version" : 2 },
		{ "position" : 1, "num_delete" : 4, "insert" : "kop", "version" : 2 }
	]
},
{
	"name" : "genstory8",
	"content" : "hello world hello world",
	"result" : "hkoptesting hello world",
	"transforms" : [
		{ "position" : 2, "num_delete" : 9, "insert" : "testing", "version" : 2 },
		{ "position" : 1, "num_delete" : 4, "insert" : "kop", "version" : 2 }
	]
},
{
	"name" : "genstory9",
	"content" : "hello world",
	"result" : "hejipkoprld",
	"transforms" : [
		{ "position" : 2, "num_delete" : 2, "insert" : "jip", "version" : 2 },
		{ "position" : 2, "num_delete" : 6, "insert" : "kop", "version" : 2 }
	]
},
{
	"name" : "genstory10",
	"content" : "hello world",
	"result" : "hjipkopld",
	"transforms" : [
		{ "position" : 3, "num_delete" : 6, "insert" : "kop", "version" : 2 },
		{ "position" : 1, "num_delete" : 2, "insert" : "jip", "version" : 2 }
	]
},
{
	"name" : "genstory11",
	"content" : "hello world",
	"result" : "hekopjippyp",
	"transforms" : [
		{ "position" : 3, "num_delete" : 0, "insert" : "jip", "version" : 2 },
		{ "position" : 2, "num_delete" : 9, "insert" : "kop", "version" : 2 },
		{ "position" : 4, "num_delete" : 7, "insert" : "pyp", "version" : 2 },
		{ "position" : 4, "num_delete" : 6, "insert" : "", "version" : 3 }
	]
},
{
	"name" : "genstory12",
	"content" : "hello world",
	"result" : "helDERYIPld",
	"transforms" : [
		{ "position" : 3, "num_delete" : 6, "insert" : "DER", "version" : 2 },
		{ "position" : 5, "num_delete" : 2, "insert" : "YIP", "version" : 2 }
	]
},
{
	"name" : "genstory13",
	"content" : "hello world",
	"result" : "helloDERYIP world",
	"transforms" : [
		{ "position" : 5, "num_delete" : 0, "insert" : "DER", "version" : 2 },
		{ "position" : 5, "num_delete" : 0, "insert" : "YIP", "version" : 2 }
	]
},
{
	"name" : "genstory14",
	"content" : "hello world",
	"result" : "hello herpy1 derpy1 herpy2 derpy2 herpy3 derpy3 herpy4 derpy4 TEST FOOBAR world",
	"flushes" : [ 3 ],
	"transforms" : [
		{ "position" : 6, "num_delete" : 0, "insert" : "herpy1 ", "version" : 2 },
		{ "position" : 13, "num_delete" : 0, "insert" : "derpy1 ", "version" : 3 },
		{ "position" : 20, "num_delete" : 0, "insert" : "herpy2 ", "version" : 4 },
		{ "position" : 27, "num_delete" : 0, "insert" : "derpy2 ", "version" : 5 },
		{ "position" : 34, "num_delete" : 0, "insert" : "herpy3 ", "version" : 6 },
		{ "position" : 41, "num_delete" : 0, "insert" : "derpy3 ", "version" : 7 },
		{ "position" : 48, "num_delete" : 0, "insert" : "herpy4 ", "version" : 8 },
		{ "position" : 55, "num_delete" : 0, "insert" : "derpy4 ", "version" : 9 },
		{ "position" : 41, "num_delete" : 0, "insert" : "TEST ", "version" : 7 },
		{ "position" : 13, "num_delete" : 0, "insert" : "FOOBAR ", "version" : 3 }
	]
},
{
	"name" : "genstory15",
	"content" : "The quick brown fox jumps over the lazy dog.",
	"result" : "The slow brown dog jumps over all lazy dogs.",
	"transforms" : [
		{ "position" : 4, "num_delete" : 5, "insert" : "slow", "version" : 2 },
		{ "position" : 30, "num_delete" : 3, "insert" : "all", "version" : 3 },
		{ "position" : 15, "num_delete" : 3, "insert" : "dog", "version" : 3 },
		{ "position" : 43, "num_delete" : 0, "insert" : "s", "version" : 2 }
	]
},
{
	"name" : "genstory16",
	"content" : "The quick brown fox jumps over the lazy dog.",
	"result" : "The slow brown * dog jumps over all lazy dogs.",
	"transforms" : [
		{ "position" : 4, "num_delete" : 5, "insert" : "slow", "version" : 2 },
		{ "position" : 30, "num_delete" : 3, "insert" : "all", "version" : 3 },
		{ "position" : 15, "num_delete" : 4, "insert" : "dog ", "version" : 3 },
		{ "position" : 14, "num_delete" : 5, "insert" : " * ", "version" : 3 },
		{ "position" : 43, "num_delete" : 0, "insert" : "s", "version" : 2 }
	]
}
] }
