## veidemannctl config import-ca

Import file with trusted certificate chains for the idp and controller.

### Synopsis

Import file with trusted certificate chains for the idp and controller. These are in addition to the default certs configured for the OS.

```
veidemannctl config import-ca FILENAME [flags]
```

### Options

```
  -h, --help   help for import-ca
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

