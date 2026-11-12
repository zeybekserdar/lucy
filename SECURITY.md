# Security Policy

## Supported versions

Security fixes are made on the `main` branch. Please run the latest version before reporting.

## Reporting a vulnerability

Please do not report security issues in public issues, discussions or pull requests.

Report them privately through GitHub instead:
[open a private security advisory](https://github.com/zeybekserdar/lucy/security/advisories/new).

Include as much of the following as you can:

- what the issue is and what an attacker could do with it,
- the steps or a proof of concept to reproduce it,
- the Lucy version or commit, and how it is deployed.

You should get a first response within a week. Once the issue is confirmed, a fix will be prepared
and released, and you will be credited in the advisory unless you prefer otherwise.

## Scope

Lucy is designed to run on a private network as a single-user app. It does not serve HTTPS itself;
exposing it to the internet without a VPN or a TLS-terminating reverse proxy is outside the
supported setup. Issues in that setup are still welcome if they are not caused by the missing TLS.
