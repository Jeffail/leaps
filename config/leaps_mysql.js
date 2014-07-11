{
	"curator": {
		"storage": {
			"type": "mysql",
			"name": "",
			"store_directory": "",
			"sql": {
				"dsn": "leaps:leaps123@tcp(localhost:3306)/leaps",
				"db_table": {
					"table": "leaps_documents",
					"id_column": "ID",
					"title_column": "TITLE",
					"description_column": "DESCRIPTION",
					"type_column": "TYPE",
					"content_column": "CONTENT"
				}
			}
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
