## veidemannctl get

Display config objects

### Synopsis

Display one or many config objects.

Valid object types:
 - browserConfig
 - browserScript
 - collection
 - crawlConfig
 - crawlEntity
 - crawlHostGroupConfig
 - crawlJob
 - crawlScheduleConfig
 - politenessConfig
 - roleMapping
 - seed

Examples:
  # List all seeds.
  veidemannctl get seed

  # List all seeds in yaml output format.
  veidemannctl get seed -o yaml

```
veidemannctl get KIND [ID ...] [flags]
```

### Options

```
  -f, --filename string      Filename to write to
  -q, --filter stringArray   Filter objects by field (i.e. meta.description=foo)
  -h, --help                 help for get
  -l, --label string         List objects by label {TYPE:VALUE | VALUE}
  -n, --name string          List objects by name (accepts regular expressions)
  -o, --output string        Output format (table|wide|json|yaml|template|template-file) (default "table")
  -p, --page int32           The page number
  -s, --pagesize int32       Number of objects to get (default 10)
  -t, --template string      A Go template used to format the output
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

