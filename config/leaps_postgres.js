{
	"curator": {
		"storage": {
			"type": "postgres",
			"name": "",
			"store_directory": "",
			"sql": {
				"dsn": "postgres://leaps:leaps123@localhost:5432/leaps?sslmode=disable",
				"db_table": {
					"table": "leaps_documents",
					"id_column": "id",
					"title_column": "title",
					"description_column": "description",
					"type_column": "type",
					"content_column": "content"
				}
			}
		}
	},
	"http_server": {
		"url": {
			"static_path": "/",
			"socket_path": "/socket",
			"address": ":8080"
		},
		"www_dir": "./static/example2"
	},
	"stats_server": {
		"static_path": "/",
		"stats_path": "/leapstats",
		"address": ":4040",
		"www_dir": "./static/stats"
	}
}
