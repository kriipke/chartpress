<!-- SPDX-License-Identifier: Apache-2.0 -->
# Third-party material and provenance

The Chartpress source, templates, documentation, and original artwork are maintained by the project copyright holder identified in `NOTICE`. Git history records one human contributor identity, Spencer Smolen (`kriipke`); automated dependency updates are authored by Dependabot.

The repository's runtime and development dependencies are not redistributed as source in this repository. Their licenses are recorded in `go.sum` and `web/package-lock.json` and checked by `make license-check`. Container images install those dependencies from their upstream distributions.

`tests/chart.zip` is a generated golden fixture produced entirely from the project-owned templates and `tests/chartpress.json`. Regenerate it deterministically with `make update-chart-fixture`; `make verify-fixtures` validates every checked-in archive.

The documentation images in `docs/`, favicon set in `docs/favicon/`, and web assets in `web/public/` and `web/src/assets/` are project assets. No third-party attribution is currently required for them. If a future asset is adapted from another source, record its source, author, and license here before committing it.

`docs/adr/_template.md` follows the widely-reused [MADR](https://adr.github.io/madr/) (Markdown Architectural Decision Records) template format, which is designed for reuse; no separate attribution is required.

`docs/principles/monorepo-polyrepo.md` was removed during the Apache-2.0 readiness review: it was a verbatim, unattributed copy of `github.com/joelparkerhenderson/monorepo-vs-polyrepo`, an upstream repository with no declared license (`"license": null` via the GitHub API). It was not referenced by any other file in the project. If similar reference material is reintroduced, link to the original instead of copying it, or copy it only with a confirmed compatible license recorded here.

Third-party dependencies retain their own copyright and license notices. Nothing in the project's Apache-2.0 license changes those terms.
