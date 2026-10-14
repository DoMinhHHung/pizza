# Pizza

**Pizza** is an early-stage project exploring a developer-friendly backend platform: a place to create and manage projects, configure environments, and eventually provision backend capabilities through a simple developer experience.

The project is currently in its foundation stage. Product scope, architecture, and implementation details will be validated incrementally; proposed components are not considered implemented until they exist in the repository.

## Goals

- Make backend project setup and management simpler for developers.
- Provide clear project and environment boundaries.
- Build a secure control plane for project configuration and lifecycle operations.
- Design long-running provisioning work to be observable, retryable, and safe to repeat.
- Keep the MVP small enough for a solo developer to build and operate.

## Planned technology

| Area | Initial direction | Notes |
| --- | --- | --- |
| Authentication | Java, Spring Boot, Spring Security | Identity, credentials, token issuance and authentication flows |
| Control plane | Go | Projects, environments, API keys, authorization checks and lifecycle operations |
| Database | PostgreSQL, initially considering Supabase | Schema and migrations must be version-controlled |
| Edge routing | Traefik | Prefer established infrastructure over a custom gateway for the MVP |
| Internal APIs | gRPC and Protocol Buffers where useful | Introduce only where the service boundary justifies it |
| Cache / rate limiting | Redis if needed | Not a mandatory dependency for the first slice |
| Deployment | Initially evaluating Render | Verify networking, gRPC support, operational limits and pricing before relying on it |

This table describes the current direction, not a claim that these technologies are already configured or deployed.

## Architecture principles

- **Start small:** add services and infrastructure only when a real requirement calls for them.
- **Clear ownership:** each service owns its behavior and data-access rules; avoid accidental cross-service coupling.
- **Security by default:** validate input, protect credentials, enforce project and organization authorization, and never trust client-supplied ownership identifiers.
- **Durable background work:** use persistent job state, idempotency, bounded concurrency, retries with backoff, and recovery for abandoned jobs. Goroutines and in-memory channels alone are not a durable queue.
- **Observable behavior:** use structured logs, health checks, actionable errors, and tests for important failure paths.
- **Incremental delivery:** implement one end-to-end feature at a time and keep the main branch buildable.

## Development status

Pizza is at the repository-foundation stage. The planned architecture and technology choices may change as requirements and deployment constraints are tested.

The initial implementation should prioritize a small, tested authentication slice, followed by the minimum control-plane flow needed to create and manage a project. Billing and more advanced platform capabilities are out of scope until the core flow is proven.

## Repository guidance

- [Implementation instructions](Instruction-Implement.md) — guidance for implementation-focused coding agents.
- [Review instructions](Instruction-Review.md) — independent code review, bug hunting and verification.
- Keep service directories and deployment configuration limited to components that are actually being implemented.
- Commit `.env.example` with fake placeholders only; never commit real credentials, tokens, private keys or production configuration.
- Keep database migrations and API contracts version-controlled.

## Engineering workflow

1. Agree on the requirement and acceptance criteria before coding.
2. Inspect the existing repository and relevant decisions before proposing changes.
3. Implement the smallest complete change that meets the requirement.
4. Run relevant formatting, tests, static analysis and build checks.
5. Review security, authorization, failure handling and maintainability.
6. Report exactly what changed and which checks were or were not run.

## Non-goals for the first MVP

- Building a custom API gateway without a demonstrated need.
- Splitting every component into a separate deployable service from day one.
- Adding Redis, Kubernetes, billing or a complex event platform solely for architectural appearance.
- Executing customer workloads inside trusted control-plane or authentication processes.

## Status

Early development — not production-ready.
