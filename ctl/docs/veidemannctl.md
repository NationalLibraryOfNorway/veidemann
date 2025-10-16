## veidemannctl

veidemannctl controls the Veidemann web crawler

### Synopsis

veidemannctl controls the Veidemann web crawler

### Options

```
      --api-key string                If set, it will be used as the bearer token for authentication
      --config string                 Path to the config file to use (By default configuration file is stored under $HOME/.veidemann/contexts/
      --context string                The name of the context to use
  -h, --help                          help for veidemannctl
      --log-caller                    include information about caller in log output
      --log-format string             set log format, available formats are: "pretty" or "json" (default "pretty")
      --log-level string              set log level, available levels are "panic", "fatal", "error", "warn", "info", "debug" and "trace" (default "info")
      --server string                 The address of the Veidemann server to use
      --server-name-override string   If set, it will override the virtual host name of authority (e.g. :authority header field) in requests
  -v, --version                       version for veidemannctl
```

### SEE ALSO

* [veidemannctl abort](veidemannctl_abort.md)	 - Abort crawl executions
* [veidemannctl abortjobexecution](veidemannctl_abortjobexecution.md)	 - Abort job executions
* [veidemannctl activeroles](veidemannctl_activeroles.md)	 - Get the active roles for the currently logged in user
* [veidemannctl completion](veidemannctl_completion.md)	 - Generate the autocompletion script for the specified shell
* [veidemannctl config](veidemannctl_config.md)	 - Modify or view configuration files
* [veidemannctl create](veidemannctl_create.md)	 - Create or update config objects
* [veidemannctl delete](veidemannctl_delete.md)	 - Delete config objects
* [veidemannctl get](veidemannctl_get.md)	 - Display config objects
* [veidemannctl import](veidemannctl_import.md)	 - Import data into Veidemann using subcommands
* [veidemannctl logconfig](veidemannctl_logconfig.md)	 - Configure logging
* [veidemannctl login](veidemannctl_login.md)	 - Log in to Veidemann
* [veidemannctl logout](veidemannctl_logout.md)	 - Log out of Veidemann
* [veidemannctl pause](veidemannctl_pause.md)	 - Request crawler to pause
* [veidemannctl report](veidemannctl_report.md)	 - Request a report
* [veidemannctl run](veidemannctl_run.md)	 - Run a crawl job
* [veidemannctl script-parameters](veidemannctl_script-parameters.md)	 - Get the effective script parameters for a crawl job
* [veidemannctl status](veidemannctl_status.md)	 - Display crawler status
* [veidemannctl unpause](veidemannctl_unpause.md)	 - Request crawler to unpause
* [veidemannctl update](veidemannctl_update.md)	 - Update fields of config objects of the same kind

