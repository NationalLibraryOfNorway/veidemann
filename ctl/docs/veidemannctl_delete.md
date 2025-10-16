## veidemannctl delete

Delete config objects

### Synopsis

Delete one or many config objects.

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
  # Delete a seed.
  veidemannctl delete seed 407a9600-4f25-4f17-8cff-ee1b8ee950f6

```
veidemannctl delete KIND ID ... [flags]
```

### Options

```
      --dry-run              Set to false to execute delete (default true)
  -q, --filter stringArray   Delete objects by field (i.e. meta.description=foo)
  -h, --help                 help for delete
  -l, --label string         Delete objects by label {TYPE:VALUE | VALUE}
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

