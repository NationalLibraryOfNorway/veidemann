## veidemannctl script-parameters

Get the effective script parameters for a crawl job

### Synopsis

Get the effective script parameters for a crawl job and optionally a seed in the context of the crawl job.

Examples:
  # See active script parameters for a Crawl Job
  veidemannctl script-parameters 5604f0cc-315d-4091-8d6e-1b17a7eb990b

  # Get effective script parameters for a Seed in the context of a Crawl Job
  veidemannctl script-parameters 5604f0cc-315d-4091-8d6e-1b17a7eb990b 9f89ca44-afe0-4f8f-808f-9df1a0fe64c9


```
veidemannctl script-parameters JOB-ID [SEED-ID] [flags]
```

### Options

```
  -h, --help   help for script-parameters
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

