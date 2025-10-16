## veidemannctl config

Modify or view configuration files

### Options

```
  -h, --help   help for config
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
* [veidemannctl config create-context](veidemannctl_config_create-context.md)	 - Create a new context
* [veidemannctl config current-context](veidemannctl_config_current-context.md)	 - Display the current context
* [veidemannctl config get-address](veidemannctl_config_get-address.md)	 - Display Veidemann controller service address
* [veidemannctl config get-apikey](veidemannctl_config_get-apikey.md)	 - Display Veidemann authentication api-key
* [veidemannctl config get-server-name-override](veidemannctl_config_get-server-name-override.md)	 - Display the server name override
* [veidemannctl config import-ca](veidemannctl_config_import-ca.md)	 - Import file with trusted certificate chains for the idp and controller.
* [veidemannctl config list-contexts](veidemannctl_config_list-contexts.md)	 - Display the known contexts
* [veidemannctl config set-address](veidemannctl_config_set-address.md)	 - Set the address to the Veidemann controller service
* [veidemannctl config set-apikey](veidemannctl_config_set-apikey.md)	 - Set the api-key to use for authentication
* [veidemannctl config set-server-name-override](veidemannctl_config_set-server-name-override.md)	 - Set the server name override
* [veidemannctl config use-context](veidemannctl_config_use-context.md)	 - Set the current context
* [veidemannctl config view](veidemannctl_config_view.md)	 - Display the current config

