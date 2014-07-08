{
        "curator": {
                "storage": {
                        "type": "memory",
                        "name": "",
                        "store_directory": "",
                        "sql": {
                                "dsn": "",
                                "db_table": {
                                        "table": "leaps_documents",
                                        "id_column": "ID",
                                        "title_column": "TITLE",
                                        "description_column": "DESCRIPTION",
                                        "content_column": "CONTENT"
                                }
                        }
                },
                "binder": {
                        "flush_period_ms": 500,
                        "retention_period_s": 60,
                        "kick_period_ms": 5
                },
                "logger": {
                        "level": 2,
                        "output_path": ""
                },
                "authenticator": {
                        "type": "none"
                }
        },
        "http_server": {
                "url": {
                        "static_path": "/leaps",
                        "socket_path": "/leaps/socket",
                        "address": ":8080"
                },
                "www_dir": "",
                "binder": {
                        "bind_send_timeout_ms": 10
                }
        },
        "stats_server": {
                "static_path": "/",
                "stats_path": "/leapstats",
                "address": ":4040",
                "www_dir": "",
                "stat_timeout_ms": 200,
                "request_timeout_s": 10
        }
}
