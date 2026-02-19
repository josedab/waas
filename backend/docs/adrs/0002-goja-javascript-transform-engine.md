# ADR-0002: goja JavaScript Engine for Payload Transforms

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS needs to allow users to transform webhook payloads before delivery. Transforms must be flexible enough to handle arbitrary JSON reshaping, field mapping, and conditional logic. The transform engine must run safely in a multi-tenant environment without compromising stability or security.

## Decision

We chose [goja](https://github.com/nicholasgasior/goja), a pure-Go JavaScript runtime, to execute user-defined payload transformations.

## Rationale

- **User familiarity:** JavaScript is the most widely understood scripting language among webhook consumers.
- **Pure Go:** No CGo dependency keeps the build simple, cross-compilable, and free of native library issues.
- **In-process execution:** Avoids the latency and operational overhead of calling an external runtime (e.g., Node.js sidecar).
- **Sandboxing:** goja scripts run with execution timeouts and restricted access to prevent runaway computations or file system access.
- **Minimal footprint:** Keeps the single-binary deployment model intact—no additional containers or runtimes required.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **Lua (gopher-lua)** | Less familiar to most webhook users; smaller ecosystem for JSON manipulation. |
| **WASM** | Higher integration complexity; requires users to compile transforms to WASM modules. |
| **Jsonnet** | Limited expressiveness—lacks loops, HTTP-aware logic, and conditional branching. |
| **External Node.js** | Adds a sidecar container, increases latency, and complicates single-binary deployments. |

## Consequences

- All transform scripts must be tested with execution timeouts to prevent runaway scripts.
- The `pkg/transform` package owns the goja runtime lifecycle and must enforce memory and CPU limits.
- Future language support (e.g., Python, WASM) can be added as additional transform engines behind the same interface.
