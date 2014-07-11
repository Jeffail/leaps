{
	"curator": {
		"storage": {
			"type": "file",
			"store_directory": "./test"
		},
		"logger": {
			"level": 3
		}
	},
	"http_server": {
		"static_path": "/",
		"socket_path": "/socket",
		"address": ":8080",
		"www_dir": "./static/example2"
	},
	"stats_server": {
		"static_path": "/",
		"stats_path": "/leapstats",
		"address": ":4040",
		"www_dir": "./static/stats"
	}
}
