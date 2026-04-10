# Specification Root

**Version:** 1.3.0  
**Status:** Active  
**Updated:** 2026-04-10

---

## Purpose

Root index for all project specifications. Each subfolder is a self-contained spec module following the [Spec Authoring Guide](./00-spec-authoring-guide/00-overview.md).

---

## Module Inventory

| # | Module | Description |
|---|--------|-------------|
| 00 | [00-spec-authoring-guide](./00-spec-authoring-guide/00-overview.md) | How to write and structure specs |
| 01 | [01-coding-guidelines](./01-coding-guidelines/00-overview.md) | Language-specific coding standards |
| 02 | [02-error-manage-spec](./02-error-manage-spec/00-overview.md) | Error handling architecture and registry |
| 03 | [03-split-db-architecture](./03-split-db-architecture/00-overview.md) | Split database architecture |
| 04 | [04-seedable-config-architecture](./04-seedable-config-architecture/00-overview.md) | Seedable configuration architecture |
| 05 | [05-design-spec](./05-design-spec/00-overview.md) | Design specifications (Mermaid styling, visual guidelines) |
| 06 | [06-diagrams](./06-diagrams/) | Mermaid flow diagrams for all CLI commands |
| 08 | [08-app](./08-app/00-overview.md) | Application project spec and documentation |
| 09 | [09-app-issues](./09-app-issues/) | Application issue tracking and resolution |
| 12 | [12-ci-cd-pipeline](./12-ci-cd-pipeline/README.md) | CI/CD pipeline: lint, test, vuln scan, build, release |
| 13 | [13-self-update-app-update](./13-self-update-app-update/README.md) | Self-update architecture, deploy strategy, release distribution |

---

## Cross-References

- [Spec Authoring Guide](./00-spec-authoring-guide/00-overview.md)
- [plan.md](../plan.md) — Task tracker
- [gitmap-v2 pipeline spec](https://github.com/alimtvnetwork/gitmap-v2/tree/main/spec/pipeline) — Reference CI/CD patterns
- [gitmap-v2 generic-update spec](https://github.com/alimtvnetwork/gitmap-v2/tree/main/spec/generic-update) — Reference self-update patterns
- [gitmap-v2 generic-release spec](https://github.com/alimtvnetwork/gitmap-v2/tree/main/spec/generic-release) — Reference release patterns
