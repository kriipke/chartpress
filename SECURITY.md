<!-- SPDX-License-Identifier: Apache-2.0 -->
# Security policy

## Supported versions

Security fixes are made on the default branch and in the latest tagged release. Older releases are not supported unless a release announcement says otherwise.

## Reporting a vulnerability

Please report vulnerabilities privately by emailing `kriipke@users.noreply.github.com`. Include the affected version or commit, impact, reproduction steps, and any suggested mitigation. Do not open a public issue until the maintainer has coordinated disclosure with you.

Never include live credentials, personal data, or production endpoints in a report or test fixture.

## Verification boundary

The required test suite uses local fakes, fake Kubernetes clients, and loopback-only HTTP test servers. It does not require production OpenAI, GitHub, S3, Kubernetes, registry, or identity-provider credentials. Those integrations require credentials only at runtime.
