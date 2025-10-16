## veidemannctl config set-address

Set the address to the Veidemann controller service

### Synopsis

Set the address to the Veidemann controller service

Examples:
  # Sets the address to Veidemann controller service to localhost:50051
  veidemannctl config set-address localhost:50051


```
veidemannctl config set-address HOST:PORT [flags]
```

### Options

```
  -h, --help   help for set-address
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

