{
	"logger": {
		"prefix": "[leaps]",
		"log_level": "DEBUG",
		"add_timestamp": true
	},
	"stats": {
		"job_buffer": 100,
		"prefix": "leaps",
		"retain_internal": true
	},
	"curator": {
		"storage": {
			"type": "mock",
			"name": "test_document"
		},
		"logger": {
			"level": "DEBUG"
		},
		"binder": {
			"transform_model": {
				"max_document_size": 5000,
				"max_transform_length": 500
			}
		}
	},
	"http_server": {
		"static_path": "/",
		"socket_path": "/socket",
		"address": ":8080",
		"www_dir": "./static/example_ace"
	},
	"stats_server": {
		"static_path": "/",
		"stats_path": "/stats",
		"address": ":4040",
		"www_dir": "./static/stats"
	}
}
