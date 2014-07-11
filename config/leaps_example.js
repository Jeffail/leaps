{
	"curator": {
		"storage": {
			"type": "mock",
			"name": "test_document"
		}
	},
	"http_server": {
		"static_path": "/",
		"socket_path": "/socket",
		"address": ":8080",
		"www_dir": "./static/example"
	},
	"stats_server": {
		"static_path": "/",
		"stats_path": "/leapstats",
		"address": ":4040",
		"www_dir": "./static/stats"
	}
}
