# Security policy

## Supported versions

Only the latest release of PicBerth receives security updates. Upgrade to the
latest release before checking whether an issue is still reproducible.

## Reporting a vulnerability

Report suspected vulnerabilities privately through
[GitHub private vulnerability reporting](https://github.com/DejavuMoe/PicBerth/security/advisories/new).
Do not disclose vulnerabilities or credentials in public issues or pull requests.

Include the affected version, deployment configuration, reproduction steps,
expected and observed behavior, and potential impact. Remove real credentials
and personal data from logs or examples. Coordinate public disclosure with the
maintainer through the private advisory.

## Dependency triage

Reviewed on 2026-11-09: `GHSA-gcqx-w48m-w2h4` affects `deepmerge@4.3.1`,
which has no patched release. PicBerth uses it only through
`@sveltejs/vite-plugin-svelte` to merge build configuration in
`src/utils/options.js` (`preResolveOptions` / `mergeConfigs`). The inputs are
plugin defaults, the repository's `web/svelte.config.js`, inline plugin
options in `web/vite.config.ts`, and Vite build settings. Uploaded files and
HTTP request data do not reach this merge; the dependency is absent from the
production application. The Dependabot alert is dismissed as not used in a
vulnerable execution path. Reassess if configuration begins accepting
untrusted data or an upstream fix becomes available. CI still audits both
runtime and development dependencies and fails on moderate or higher findings.

The bundled design export tools are development dependencies too: keep their
lockfiles updated and run their builds, tests, and audits in CI.
