## veidemannctl logout

Remove locally stored Veidemann credentials

### Synopsis

Remove the current context's locally stored ID and refresh tokens.

This does not revoke tokens or end sessions in Dex, Keycloak, or an upstream
identity provider. A copied refresh token may remain usable until it expires or
is revoked by the provider.

```
veidemannctl logout [flags]
```

### Options

```
  -h, --help   help for logout
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

* [veidemannctl](veidemannctl.md)	 - veidemannctl controls the Veidemann web crawler

