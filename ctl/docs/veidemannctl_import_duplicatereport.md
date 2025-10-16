## veidemannctl import duplicatereport

List duplicated seeds or crawl entities in Veidemann

```
veidemannctl import duplicatereport KIND [flags]
```

### Options

```
  -b, --db-dir string     Directory for storing state db (default "/tmp/veidemannctl")
  -h, --help              help for duplicatereport
      --ignore-scheme     Ignore the URL's scheme when checking for duplicates.
  -o, --out-file string   File to write output.
      --skip-import       Do not import existing seeds into state database
      --toplevel          Convert URI to toplevel by removing path before checking for duplicates.
      --truncate          Truncate state database
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

* [veidemannctl import](veidemannctl_import.md)	 - Import data into Veidemann using subcommands

