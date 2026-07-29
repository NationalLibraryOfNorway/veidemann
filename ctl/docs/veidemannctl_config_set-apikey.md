## veidemannctl config set-apikey

Set the api-key to use for authentication

### Synopsis

Set the api-key to use for authentication

Examples:
  # Enter an API key without echoing it
  veidemannctl config set-apikey

  # Read an API key from a secret manager
  secret-tool lookup service veidemann | veidemannctl config set-apikey

```
veidemannctl config set-apikey [flags]
```

### Options

```
  -h, --help   help for set-apikey
```

### Options inherited from parent commands

```
      --config string                 Path to the config file to use (By default configuration file is stored under $HOME/.veidemann/contexts/
      --context string                The name of the context to use
      --insecure                      If set, it will use an insecure connection
      --log-caller                    include information about caller in log output
      --log-format string             set log format, available formats are: "pretty" or "json" (default "pretty")
      --log-level string              set log level, available levels are "panic", "fatal", "error", "warn", "info", "debug" and "trace" (default "info")
      --server string                 The address of the Veidemann server to use
      --server-name-override string   If set, it will override the virtual host name of authority (e.g. :authority header field) in requests
```

### SEE ALSO

* [veidemannctl config](veidemannctl_config.md)	 - Modify or view configuration files

