## veidemannctl completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	veidemannctl completion fish | source

To load completions for every new session, execute once:

	veidemannctl completion fish > ~/.config/fish/completions/veidemannctl.fish

You will need to start a new shell for this setup to take effect.


```
veidemannctl completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
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

* [veidemannctl completion](veidemannctl_completion.md)	 - Generate the autocompletion script for the specified shell

