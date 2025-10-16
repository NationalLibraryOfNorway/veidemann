## veidemannctl report crawllog

View crawl log

### Synopsis

View crawl log.

```
veidemannctl report crawllog [ID ...] [flags]
```

### Options

```
      --execution-id string   Execution ID
  -f, --filename string       Filename to write to
  -h, --help                  help for crawllog
  -o, --output string         Output format (table|wide|json|yaml|template|template-file) (default "table")
  -p, --page int32            The page number
  -s, --pagesize int32        Number of objects to get (default 10)
  -t, --template string       A Go template used to format the output
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

* [veidemannctl report](veidemannctl_report.md)	 - Request a report

