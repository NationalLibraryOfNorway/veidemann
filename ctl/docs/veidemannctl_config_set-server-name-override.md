## veidemannctl config set-server-name-override

Set the server name override

### Synopsis

Set the server name override.

Use this when there is a mismatch between the exposed server name for the cluster and the certificate. The use is a security
risk and is only recommended for testing.

Examples:
  # Sets the server name override to test.local
  veidemannctl config set-server-name-override test.local


```
veidemannctl config set-server-name-override HOSTNAME [flags]
```

### Options

```
  -h, --help   help for set-server-name-override
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

* [veidemannctl config](veidemannctl_config.md)	 - Modify or view configuration files

