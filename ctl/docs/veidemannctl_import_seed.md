## veidemannctl import seed

Import seeds

### Synopsis

Import new seeds and entities from a line oriented JSON file on the following format: 

{"entityName":"foo","uri":"https://www.example.com/","entityDescription":"desc","entityLabel":[{"key":"foo","value":"bar"}],"seedLabel":[{"key":"foo","value":"bar"},{"key":"foo2","value":"bar"}],"seedDescription":"foo"}

Every record must be formatted on a single line.




```
veidemannctl import seed [flags]
```

### Options

```
      --check-uri                    Check the uri for liveness and follow permanent redirects
      --check-uri-timeout duration   Timeout duration when checking uri for liveness (default 2s)
  -c, --concurrency int              Number of concurrent workers (default 16)
      --crawljob-id string           Set crawlJob ID for new seeds
  -b, --db-dir string                Directory for storing state db (default "/tmp/veidemannctl")
      --dry-run                      Run without actually writing anything to Veidemann
  -e, --err-file string              File to write errors to. "-" writes to stderr (default "-")
  -f, --filename string              Filename or directory to read from. If input is a directory, all files ending in .yaml or .json will be tried. An input of '-' will read from stdin.
  -h, --help                         help for seed
      --ignore-scheme                Ignore the URL's scheme when checking if this URL is already imported (default true)
      --skip-import                  Do not import existing seeds into state database
      --toplevel                     Convert URI by removing path
      --truncate                     Truncate state database
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

