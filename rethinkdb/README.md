# RethinkDB container startup

`rethinkdb-startup`, built from the standard-library-only Go module in `startup/`,
discovers join targets through the Kubernetes headless Service and then replaces
itself with RethinkDB using `syscall.Exec`. It does not probe the cluster port with raw TCP:
RethinkDB performs the clustering handshake and retries unavailable peers itself.
The health probe remains a separate binary and module.

Startup logs use the standard `log/slog` text handler on stderr. Pod identity,
service, attempt counts, and discovered peers appear as key-value fields.
Lookup failures and unverified bootstrap use WARN; terminal startup failures use
ERROR. Passwords and full command lines are never logged. Once startup replaces
itself with RethinkDB, the database retains its own logging format.

## Discovery and bootstrap

Explicit `RETHINKDB_SEEDS` take precedence over automatic discovery. Otherwise,
pods whose names end in a numeric ordinal query
`RETHINKDB_SERVICE_NAME.POD_NAMESPACE.svc.RETHINKDB_CLUSTER_DOMAIN` with
Go's DNS resolver. Successful responses supply unique addresses other than
`POD_IP` as `--join` targets. Automatic StatefulSet discovery accepts a response
only if it also contains the current `POD_IP`. Responses that omit this pod are
retried, even if they contain other peers: they can be stale during a rollout.
Discovery stops when an accepted response supplies peers, even if those peers
have not opened their cluster listeners yet.

Lookups use context deadlines; SIGTERM or SIGINT cancels discovery and retry
delays promptly. The static binary uses Go's resolver and the container's DNS
configuration, without `getent`, libc NSS plugins, or a timeout subprocess.
Temporary errors in either the A or AAAA lookup fail the combined lookup rather
than accepting partial results. Failed lookups, including timeouts, discard all output;
they never establish that the cluster is empty. Each attempt logs its result.
After all attempts, only ordinal `0` in server mode may start without joins,
and only if the final lookup succeeded and included its own `POD_IP` with no
other addresses. An empty successful response does not qualify. An earlier
successful self-only response does not qualify if the final lookup fails.
All other discovery exhaustion exits nonzero for Kubernetes to restart the
container and repeat discovery.

Bootstrap means starting without joins; it does not erase existing data. A new
cluster starts with ordinal 0, and subsequent members join it. Higher ordinals
cannot bootstrap if ordinal 0 and all other members are unavailable. A restarting
ordinal 0 joins discovered members instead of automatically starting alone.
Self-only DNS is bootstrap eligibility, not proof that no other cluster exists:
an incomplete DNS response or incorrect Service selector can still hide peers.

Ordinal-less pods, including the admin proxy, use the headless Service hostname
as their default join target, so they do not require ordinal 0 specifically.
The Service must select server members, not proxies, and retain
`publishNotReadyAddresses: true` so concurrently starting members can find each
other. `POD_IP` must match the pod address returned by discovery; Kubernetes
manifests supply it through the Downward API.

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `POD_NAME` | Required | Server name source and numeric ordinal detection. |
| `POD_IP` | `127.0.0.1` | Advertised cluster address and self-address required in automatic discovery; supply the actual pod IP in Kubernetes. |
| `POD_NAMESPACE` | `default` | Namespace containing the headless Service. |
| `RETHINKDB_SERVICE_NAME` | `rethinkdb` | Headless Service used for discovery. |
| `RETHINKDB_CLUSTER_DOMAIN` | `cluster.local` | Kubernetes DNS cluster domain. |
| `RETHINKDB_CLUSTER_PORT` | `29015` | Listener port, canonical port, and default seed port. |
| `RETHINKDB_SEEDS` | Empty | Explicit join endpoints, separated by commas or whitespace, including newlines and tabs. |
| `RETHINKDB_DISCOVERY_ATTEMPTS` | `5` | Maximum automatic DNS lookup attempts. |
| `RETHINKDB_DISCOVERY_DELAY_SECONDS` | `2` | Delay between unsuccessful discovery attempts, with no final sleep. |
| `RETHINKDB_DISCOVERY_DNS_TIMEOUT_SECONDS` | `2` | Context deadline for each automatic DNS lookup. |
| `RETHINKDB_PASSWORD` | `auto` | Forwarded as the initial password. Kubernetes supplies the configured password through a Secret. |
| `PROXY` | Unset | Any nonempty value selects proxy mode, which requires join targets. |

Attempts and DNS timeout must be positive decimal integers; delay may be zero.
These settings are limited to `2147483647` to bound integer arithmetic. Empty,
malformed, negative, and out-of-range values fail with a configuration error.
Leading zeros are accepted and normalized as decimal, not octal.

An unset or empty seed value selects automatic discovery. A nonempty value made
only of separators is an error. Seeds accept hostnames, IPv4, and raw or bracketed
IPv6; use `[IPv6]:port` to specify an IPv6 endpoint's port. Explicit seed ports
are preserved and validated as integers in `1..65535`, as is the listener port.

Both `--cluster-port VALUE` and `--cluster-port=VALUE` are accepted. The entrypoint
passes exactly one normalized listener option to RethinkDB. A CLI value replaces
the default, but must agree with `RETHINKDB_CLUSTER_PORT` if that environment
variable was explicitly set. Repeated CLI port options are errors. Other
arguments are forwarded unchanged. Configure the cluster port through this
entrypoint interface rather than relying on a RethinkDB config-file port: the
entrypoint always supplies the effective listener port on the command line.

Custom cluster ports also require corresponding overlay changes to server and
proxy container ports, Service ports, and applicable network policies or mesh
configuration. Explicit seeds may use a different remote port. The driver port
and the probe's `RETHINKDB_URL` are separate settings.

`RETHINKDB_STATEFULSET_NAME` remains accepted and logged for compatibility, but
does not choose the ordinal-less seed anymore. `RETHINKDB_DISCOVERY_CONNECT_TIMEOUT_SECONDS`
is obsolete and unused; the DNS timeout setting does not open a cluster socket.

## Recovery and Kubernetes probes

Every member advertises `POD_IP:RETHINKDB_CLUSTER_PORT` as its canonical address,
including StatefulSet members. RethinkDB 2.4.4 resolves advertised hostnames
during cluster handshakes and retains resolved addresses for reconnection. When
StatefulSet DNS is stale during replacement, hostname canonical addresses can
therefore leave a persistent missing connection even after a server is ready.
Advertising the current pod IP removes DNS from canonical address exchange and
routing to newly introduced peers. Persistent server IDs and data still come
from the existing data directory; a stable hostname is not the server identity.

DNS remains the seed-discovery mechanism; Kubernetes API access, RBAC, and new
service accounts are not required. Seeing this pod in a DNS response does not
prove that every other address is fresh. RethinkDB can learn current numeric
canonical addresses through any peer it successfully joins. If all initial
targets are stale or unreachable, startup recovery still depends on restart.

The default automatic discovery budget is approximately 18 seconds:
five two-second lookups plus four two-second delays, plus scheduling overhead.
There is no subprocess termination grace period. Explicit seeds and ordinal-less
hostname joins bypass this lookup loop.

Initial join addresses are a snapshot. Native connection retries handle a peer
that becomes available at its discovered address; they do not refresh this
entrypoint's discovery after a pod is replaced at a different IP. RethinkDB opens
its cluster listener before waiting for initial joins, allowing concurrent
startup, but the driver endpoint is not available while that initial join waits.

Server and admin startup probes run every ten seconds, allow five seconds per
probe, and restart after ten failures: an approximate 100-second startup window,
not a precise deadline. Restarting reruns discovery or resolves configured join
hostnames again. DNS errors in native startup may also cause an earlier exit.
Increase the startup-probe budget in overlays if discovery settings or expected
database startup time increase. Outside Kubernetes, provide an equivalent restart
policy and startup watchdog if automatic stale-address recovery is required.

After startup, RethinkDB manages cluster connections. Readiness and liveness are
local driver checks, not verification of expected membership, replication, or
all-to-all connectivity. Ensure `RETHINKDB_URL` reaches the configured driver
listener and the probe has the configured password. Persistent DNS or network
failures should remain visible in entrypoint logs, RethinkDB logs, and Kubernetes
probe failures; this entrypoint does not suppress those diagnostics.

Removing raw TCP probes addresses misleading `invalid clustering header` warnings
only. Numeric canonical addresses additionally address the reproduced
`non_transitive_error` caused by stale canonical-hostname resolution. They do not
hide genuine network partitions, filtering, or other cluster connectivity errors.

## Validation and rollout checklist

Run tests and static validation from the repository root:

```sh
go -C rethinkdb/startup test -race ./...
go -C rethinkdb/startup vet ./...
go -C rethinkdb/probe test ./...
git diff --check
kustomize build deploy/k8s/base/rethinkdb > /dev/null
kustomize build deploy/k8s/overlays/dev/rethinkdb > /dev/null
kustomize build deploy/k8s/overlays/dev/rethinkdb-backup > /dev/null
```

Build and publish a new container image, then select its immutable tag in the
deployment overlay alongside these manifest changes. The checked-in base image
tag is not automatically updated by editing `startup/`. Validate in a disposable
environment before rolling out to an existing cluster; do not delete production
PVCs to exercise bootstrap.

Roll out the new image one member at a time, preserving every PVC. Check
`server_status.network.canonical_addresses` for the current pod IP on each
updated member and verify full `connected_to` connectivity before continuing.
Old members continue advertising hostnames until they are replaced, so the first
rollout can still encounter stale hostname resolution involving an old member.
Complete the migration and verify a subsequent rollout; local driver readiness
alone does not establish full cluster connectivity. No new image tag is selected
automatically by these source changes.

- Start one fresh member, then add members incrementally. Also start several
  members concurrently. Verify that they converge into one cluster.
- Restart ordinal 0 while another member remains healthy; verify that it joins
  that member and retains its data.
- Inject DNS failures, empty successful responses, and self-only responses.
  Only ordinal 0 with a final successful self-only response may bootstrap.
  Check a self-only response followed by a final failure and the reverse order.
  Check that partial output on resolver failure or timeout is ignored.
- Return other peers plus this pod's previous IP from headless DNS. Verify that
  startup retries until the response includes its current IP, or exits after
  exhaustion; it must not accept the stale peers or bootstrap from that response.
- On one member, pin another member's canonical hostname to a stale IP while
  leaving headless discovery correct (for example, using a test-container hosts
  entry). With the old hostname canonical addresses this reproduces a missing
  connection and `non_transitive_error`. With numeric canonical addresses, verify
  full membership and mutual connectivity despite the same stale hostname entry.
  Repeat pod replacements at new IPs while retaining their data volumes.
- Delay a peer's listener at the same IP; verify a native join succeeds. Then
  supply stale addresses and replace the peer at a new IP; verify a startup-probe
  restart reruns discovery and eventually joins the replacement.
- Exercise multiline seeds, separator-only seeds, duplicates, IPv4, IPv6,
  leading zeros, invalid or overflowing settings, and both CLI port forms.
  Verify conflicting or repeated port flags fail and custom listener/canonical
  ports agree while explicit remote seed ports are preserved.
- Start the admin proxy with ordinal 0 unavailable and another member healthy.
  Verify it joins through the headless Service without an init-container probe.
- Inspect `rethinkdb.server_status`, its connectivity information, and
  `rethinkdb.current_issues` to verify membership and connectivity. Check genuine
  failure diagnostics remain visible. Warning absence alone is not acceptance.

Go tests cover configuration and seed parsing, bootstrap eligibility, partial
DNS failures, deadlines, cancellation, and process replacement with preserved
arguments, environment, PID, and RethinkDB exit status. They use only the standard
library and do not require a database or shell test harness.

Unit tests and static checks do not establish runtime cluster convergence. Record which of these
integration scenarios were actually exercised when handing off a deployment.
