{
        "curator": {
                "storage": {
                        "type": "mock",
                        "name": "test_document"
                },
                "binder": {
                        "flush_period_ms": 500,
                        "retention_period_s": 60,
                        "kick_period_ms": 5
                },
                "logger": {
                        "level": 2,
                        "output_path": ""
                }
        },
        "http_server": {
                "url": {
                        "static_path": "/",
                        "socket_path": "/socket",
                        "address": ":8080"
                },
                "www_dir": "./static/example",
                "binder": {
                        "bind_send_timeout_ms": 10
                }
        },
        "stats_server": {
                "static_path": "/",
                "stats_path": "/leapstats",
                "address": ":4040",
                "www_dir": "./static/stats",
                "stat_timeout_ms": 200,
                "request_timeout_s": 10
        }
}
