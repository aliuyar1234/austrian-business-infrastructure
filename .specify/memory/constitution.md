<!--
SYNC IMPACT REPORT
==================
Version change: N/A (initial) → 1.0.0
Modified principles: N/A (initial constitution)
Added sections:
  - Core Principles (9 principles in 4 categories)
  - Constraints (Testing, Dependencies)
  - Quality Gates (Readability, Structure, Spaghetti Prevention)
  - Governance
Removed sections: N/A
Templates requiring updates:
  - .specify/templates/plan-template.md: ✅ Compatible (Constitution Check section exists)
  - .specify/templates/spec-template.md: ✅ Compatible (scope alignment implicit)
  - .specify/templates/tasks-template.md: ✅ Compatible (test-first workflow exists)
Follow-up TODOs: None
-->

# Austrian Business Infrastructure Constitution

## Core Principles

### I. Delete Over Add

Code volume MUST be minimized. No code without proven necessity.

- If removing code does not break tests, the code MUST be removed
- Every addition MUST demonstrate clear, immediate value
- Default action is deletion; addition requires justification

**Rationale**: LLMs tend to generate excessive code. This principle counteracts that tendency by making deletion the default and requiring proof for additions.

### II. Deferred Abstraction

Abstraction MUST NOT be introduced until a pattern has been observed at least 3 times.

- No "just in case" flexibility
- Duplicate code is acceptable until the third occurrence
- Premature generalization is prohibited

**Rationale**: Early abstraction creates unnecessary complexity and often targets the wrong axis of variation. Waiting for concrete evidence ensures abstractions solve real problems.

### III. Strict Scope Adherence

Implementation MUST build exactly what the specification states. Nothing more.

- "Nice to have" features go to backlog, not current work
- Scope creep is a defect, not initiative
- Every addition not in spec requires explicit approval

**Rationale**: LLMs often add unrequested features. Strict scope prevents bloat and keeps deliverables predictable.

### IV. Test-First Development

Tests MUST fail before implementation begins.

- Write test → verify it fails → implement → verify it passes
- Integration tests with real database preferred over mocks
- Mocks ONLY permitted for external APIs without sandbox environments

**Rationale**: Tests written after implementation often test the implementation rather than the requirement. Failing tests first proves the test actually validates behavior.

### V. Minimal Dependencies

Every new dependency MUST be justified.

- Standard library preferred over external packages
- For any proposed dependency, document:
  - Why stdlib is insufficient
  - Whether actively maintained
  - Count of transitive dependencies

**Rationale**: Dependencies are liabilities. Each one adds supply chain risk, update burden, and potential breaking changes.

### VI. Obvious Over Clever

Code MUST be immediately understandable without comments.

- If code requires a comment to explain what it does, rewrite the code
- Comments MAY explain why, never what
- One function = one job
- Names describe what something does, not how

**Rationale**: Clever code impresses during writing but costs during reading. Code is read far more than written.

### VII. Single Responsibility Structure

Each module MUST have exactly one responsibility.

- No circular dependencies
- Data flows in one direction
- Entry point MUST be obvious with no magic initialization
- Dependencies MUST be explicit (injection), not hidden (global state, service locators)

**Rationale**: Clear module boundaries make code navigable and testable. Hidden dependencies cause surprising failures.

### VIII. Shallow Nesting

Code MUST NOT exceed 3 levels of nesting. Deeper nesting requires refactoring.

- Extract functions rather than indent
- No chain of functions calling functions for the same data transformation
- Early returns preferred over nested conditionals

**Rationale**: Deep nesting obscures logic flow and makes code difficult to trace mentally.

## Constraints

### Testing Constraints

| Constraint | Rule |
|------------|------|
| Test timing | Tests MUST exist and fail before implementation |
| Test type preference | Integration tests with real DB over mocks |
| Mock usage | ONLY for external APIs lacking sandbox environments |
| Test coverage | Tests MUST cover spec requirements, not implementation details |

### Dependency Constraints

| Constraint | Rule |
|------------|------|
| New dependency | MUST include written justification |
| Preference order | stdlib → well-maintained minimal-dep package → larger framework |
| Evaluation criteria | Active maintenance, transitive dependency count, license |
| Go-specific | Use stdlib `net/http`, `encoding/xml`, etc. before external packages |

## Quality Gates

### Code Review Gates

All pull requests MUST verify:

1. No code added without spec requirement
2. No abstractions without 3+ concrete uses
3. No nesting deeper than 3 levels
4. No hidden dependencies
5. No circular imports/dependencies
6. All tests written before implementation (commit history evidence)
7. New dependencies justified in PR description

### Pre-Merge Checklist

- [ ] Removing any code does not leave tests passing (tests validate behavior)
- [ ] All new dependencies documented with justification
- [ ] No comments explaining what code does (only why if needed)
- [ ] Data flow is unidirectional within module
- [ ] Entry points are explicit and obvious

## Governance

This constitution supersedes all other development practices for the Austrian Business Infrastructure project.

### Amendment Procedure

1. Propose amendment with rationale
2. Document migration plan for existing code if principle changes
3. Obtain approval from project maintainers
4. Update this document with new version
5. Propagate changes to dependent templates

### Versioning Policy

- **MAJOR**: Backward-incompatible principle removal or redefinition
- **MINOR**: New principle added or existing principle materially expanded
- **PATCH**: Clarifications, wording improvements, non-semantic refinements

### Compliance Review

- All PRs MUST include constitution compliance verification
- Complexity beyond these principles MUST be justified in writing
- Violations require explicit exception approval with documented reasoning

**Version**: 1.0.0 | **Ratified**: 2025-12-07 | **Last Amended**: 2025-12-07
