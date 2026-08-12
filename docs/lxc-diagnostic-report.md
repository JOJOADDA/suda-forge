# SUDA FORGE — Phase 1 LXC Diagnostic Report

**Date:** 2026-08-12  
**Status:** Phase 1 remains open; LXC integration is blocked by the current nested execution environment.

## A. Current environment

The environment is Ubuntu 24.04.4 LTS on `x86_64`, running kernel `6.1.102` as user `ubuntu` with passwordless sudo. The process environment is itself detected as Docker: `systemd-detect-virt` returns `docker`, PID 1 is in the container cgroup, and the user-facing filesystem exposes a cgroup v2 mount.

LXC userspace 5.0.3 is installed, but the `lxc`/LXD command is not installed. The classic LXC tools are present, including `lxc-create`, `lxc-start`, `lxc-stop`, `lxc-info`, `lxc-attach`, and `lxc-destroy`. PostgreSQL, Go, and the frontend toolchain are available. The system has approximately 31 GB free disk and 2.8 GB available memory.

| Area | Observed result |
|---|---|
| OS | Ubuntu 24.04.4 LTS |
| Kernel | Linux 6.1.102, x86_64 |
| Current user | `ubuntu`, UID 1000 |
| Root/sudo | `sudo -n true` succeeds |
| Virtualization | `docker` |
| LXC | Classic LXC 5.0.3; no `lxc`/LXD client |
| User mappings | `ubuntu:100000:65536` in `/etc/subuid` and `/etc/subgid` |
| Disk | 40 GB filesystem, approximately 31 GB available |
| Memory | 3.8 GiB total, approximately 2.8 GiB available |
| Firewall | iptables policies observed as ACCEPT; UFW is unavailable in this sandbox |
| Proxy | Only OpenAI proxy variables are present; no general HTTP(S) proxy |

The full raw evidence is stored in `tests/lxc-environment-diagnostic.txt`, `tests/lxc-failure-reproduction.txt`, `tests/lxc-nesting-capabilities.txt`, and the individual startup logs.

## B. Exact LXC failure

The initial report that the image download failed was incomplete. The direct LXC operation produced:

```text
Downloading the image index
Downloading the rootfs
```

and appeared stalled because the Ubuntu Jammy rootfs is approximately **135,030,784 bytes** and the selected mirror transferred a 1 MiB range at approximately **232 KiB/s**. The exact image URL was resolved successfully, and TLS validation succeeded. When allowed to run for the required duration, the real image creation completed:

```text
The image cache is now ready
Unpacking the rootfs
You just created an Ubuntu jammy amd64 (20260810_07:42) container.
```

The image download was therefore slow, not a DNS/TLS/image-alias failure.

The first privileged container was intentionally not accepted because `lxc-ls` reported `UNPRIVILEGED false`, which violates SUDA FORGE’s security requirement. A user-owned container was then created with the documented idmap:

```text
lxc.idmap = u 0 100000 65536
lxc.idmap = g 0 100000 65536
```

Its startup failed with the following exact errors:

```text
Operation not permitted - Failed to allocate new network namespace id
Failed to open "/etc/lxc/lxc-usernet"
Permission denied - The cgroup.threads file is not writable, skipping unified hierarchy
```

After the network definition was temporarily removed for diagnosis, startup progressed further but failed at the required mount boundary:

```text
Operation not permitted - Failed to allocate new network namespace id
Operation not permitted - Failed to mount "sysfs"
Failed to setup first automatic mounts
```

## C. Root cause

There are two separate findings.

First, the apparent image-download failure was caused by a slow redirected mirror. The network, DNS, HTTPS, certificate chain, and image metadata all worked. A longer bounded timeout is required for first-time image provisioning.

Second, the real unprivileged container cannot start inside the current Docker-backed sandbox with the required project-computer features. The environment lacks `/etc/lxc/lxc-usernet`, the LXC service and LXCFS service are inactive because this environment does not provide a normal host init boundary, and the cgroup v2 hierarchy is not delegated to the unprivileged user. More decisively, direct tests show that the `ubuntu` user cannot independently perform `unshare -n` or a private mount namespace operation; both return `Operation not permitted`. LXC therefore cannot reliably create the network namespace and mount the container’s sysfs under the required unprivileged execution model.

The direct root tests are not sufficient evidence of support: root can perform those operations, but SUDA FORGE must not use privileged project containers or root agents to bypass the unprivileged boundary.

## D. Whether LXC is supported in this environment

**Image creation is supported. A compliant SUDA FORGE unprivileged Project Computer is not supported in the current environment.** The environment is itself a restricted Docker container rather than a normal Linux host or VM with delegated LXC capabilities. A privileged LXC container could be created, but it was rejected and destroyed because it would violate the specification.

## E. Fix applied

No host security mechanism was disabled. AppArmor was not permanently disabled. The firewall was not changed. The system was not converted to privileged containers. No Docker fallback or fake runtime was introduced.

The only reversible configuration used during diagnosis was the per-user LXC idmap file at `~/.config/lxc/default.conf`, which is the standard configuration suggested by LXC for an unprivileged user. Diagnostic containers and temporary directories were removed, and `/home/ubuntu` was restored to mode `750` after a temporary traverse-only test.

In the SUDA FORGE code, the LXC adapter was corrected to initialize `/workspace` only after the container has actually started, to support an explicit LXC path using `-P`, to preserve command output in errors, and to stop reporting a runtime as created when workspace initialization or lifecycle operations fail. A deterministic `RuntimeProvider` contract harness was added for tests; it is test-only and is not used as a production runtime.

## F. Real LXC test result

The real image download and rootfs unpack test passed. A compliant unprivileged lifecycle test did **not** pass. The privileged container test was deliberately rejected. The unprivileged container was created, but `lxc-start` failed before a usable container became `RUNNING`; therefore no claim is made that terminal, filesystem, Git, processes, port discovery, or preview passed inside LXC.

## G. Components already passing

The following components pass independently:

| Component | Result |
|---|---|
| Go backend compilation | Pass |
| Go unit tests | Pass |
| Project slug and lifecycle state-machine tests | Pass |
| Filesystem traversal rejection tests | Pass |
| RuntimeProvider deterministic contract harness | Pass |
| PostgreSQL schema initialization | Pass |
| Live `/healthz` endpoint | Pass |
| Live project listing endpoint | Pass |
| React TypeScript production build | Pass |
| Frontend project dashboard API wiring | Pass |
| LXC image metadata and HTTPS access | Pass |
| LXC image download and unpack | Pass |

The repository is at `/home/ubuntu/suda-forge`, with milestone commit `3bac2a2` followed by the current diagnostic/code-review changes.

## H. Components still blocked

The following must remain explicitly blocked until a suitable Linux host or VM is available: real unprivileged project lifecycle, persistent runtime filesystem validation, real PTY terminal inside LXC, Git inside the project computer, process management inside the project computer, listening-port discovery, Caddy/preview routing, and the complete acceptance test.

The current terminal package contains the provider-neutral boundary but not yet the production PTY/WebSocket implementation. This is intentionally not marked complete.

## I. Security compromises

**None.** No privileged container was accepted as a substitute. No host shell or host filesystem was exposed to the application. No AppArmor disablement, firewall bypass, root agent, Docker fallback, or fake LXC path was introduced.

## J. Exact next action required

The next engineering action is to run the same Phase 1 acceptance test on a normal Linux host or VM where the SUDA FORGE service account can create unprivileged LXC containers with delegated user namespaces, cgroup v2 delegation, LXC network configuration, and the required mount operations. The target must provide:

1. A real Linux host or VM, not a restricted Docker container.
2. LXC 5.x or LXD with an unprivileged container profile.
3. Configured `/etc/subuid` and `/etc/subgid` ranges for the service account.
4. Delegated cgroup v2 controls for that account or its service scope.
5. A configured LXC bridge or equivalent controlled network path and `/etc/lxc/lxc-usernet` policy.
6. AppArmor/seccomp enabled with a profile permitting ordinary unprivileged containers.
7. Sufficient disk for base images and project files, and outbound HTTPS access to the configured image source.
8. No requirement to grant the project runtime host root access.

Until that environment is available, SUDA FORGE remains in **Phase 1** and the AI orchestration, model registry, agent council, autonomous repair, and advanced deployment work must not begin.
