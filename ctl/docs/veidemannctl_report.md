## veidemannctl report

Request a report

### Options

```
  -h, --help   help for report
```

### Options inherited from parent commands

```
      --api-key string                If set, it will be used as the bearer token for authentication
      --config string                 Path to the config file to use (By default configuration file is stored under $HOME/.veidemann/contexts/
      --context string                The name of the context to use
      --log-caller                    include information about caller in log output
      --log-format string             set log format, available formats are: "pretty" or "json" (default "pretty")
      --log-level string              set log level, available levels are "panic", "fatal", "error", "warn", "info", "debug" and "trace" (default "info")
      --server string                 The address of the Veidemann server to use
      --server-name-override string   If set, it will override the virtual host name of authority (e.g. :authority header field) in requests
```

### SEE ALSO

* [veidemannctl](veidemannctl.md)	 - veidemannctl controls the Veidemann web crawler
* [veidemannctl report crawlexecution](veidemannctl_report_crawlexecution.md)	 - Get current status for crawl executions
* [veidemannctl report crawllog](veidemannctl_report_crawllog.md)	 - View crawl log
* [veidemannctl report jobexecution](veidemannctl_report_jobexecution.md)	 - Get current status for job executions
* [veidemannctl report pagelog](veidemannctl_report_pagelog.md)	 - View page log
* [veidemannctl report query](veidemannctl_report_query.md)	 - Run a database query

