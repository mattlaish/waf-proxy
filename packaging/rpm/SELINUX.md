# SELinux boundary for the RHEL-family package

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](../../DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](../../SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](../../TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Slice B uses standard RHEL filesystem locations (`/usr`, `/etc`, `/var/lib`,
`/var/log`, `/run`) and does not install an ad-hoc SELinux policy module or run
`semanage`/`setenforce` from package scriptlets.

This is deliberate:

- package installation must not weaken or disable SELinux;
- custom local policy must not be generated from transient AVCs automatically;
- clean-host qualification under enforcing SELinux belongs to Enterprise Linux
  Distribution Packaging Slice D;
- if qualification demonstrates a project-owned policy is required, it must be
  developed and versioned explicitly rather than injected from `%post`.

The systemd units continue to use capability bounding and filesystem sandboxing.
Any RHEL-family AVC observed during Slice D is release evidence and must be
resolved before that distribution is promoted from NOT_RUN/DEFERRED.
