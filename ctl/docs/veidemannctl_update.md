## veidemannctl update

Update fields of config objects of the same kind

### Synopsis

Update a field of one or many config objects of the same kind

```
veidemannctl update KIND [ID ...] [flags]
```

### Examples

```
# Add CrawlJob for a seed.
veidemannctl update seed -n "https://www.gwpda.org/" -u seed.jobRef+=crawlJob:e46863ae-d076-46ca-8be3-8a8ef72e709

# Replace all configured CrawlJobs for a seed with a new one.
veidemannctl update seed -n "https://www.gwpda.org/" -u seed.jobRef=crawlJob:e46863ae-d076-46ca-8be3-8a8ef72e709
```

### Options

```
  -q, --filter stringArray    Filter objects by field (i.e. meta.description=foo)
  -h, --help                  help for update
  -l, --label string          Filter objects by label {TYPE:VALUE | VALUE}
  -s, --limit int32           Limit the number of objects to update. 0 = no limit
  -n, --name string           Filter objects by name (accepts regular expressions)
  -u, --update-field string   Which field to update (i.e. meta.description=foo)
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

