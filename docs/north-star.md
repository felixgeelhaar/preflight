# Preflight North Star Metric

## NSM: Time-to-First-Successful-Apply (TTFSA)

**Definition.** Median wall-clock time, in minutes, between the first
successful invocation of `preflight init` and the first invocation of
`preflight apply` that exits with status `0` and a green `preflight doctor`.

**Target.** ≤ 10 minutes for the wedge persona ("set up a new Mac").

**Why this metric.**

- It directly measures the killer loop: capture/init → plan → apply → doctor.
- It is sensitive to friction at every step that matters: install path,
  quickstart correctness, plan readability, error recoverability, doctor
  noise.
- It cannot be gamed by shipping more features; only by removing friction.

**Anti-NSMs (do not optimize).**

- Number of providers (already too many — see CLI grouping work).
- Number of stars / GitHub vanity metrics.
- Lines of YAML supported (more surface = more configuration burden).

## Activation events (opt-in only)

The minimum viable telemetry pipeline records four events:

| Event                 | Fires when                                             |
|-----------------------|--------------------------------------------------------|
| `init.completed`      | `preflight init` writes `preflight.yaml` for the first time |
| `apply.first_success` | First `preflight apply` exits 0 with at least one applied step |
| `doctor.green`        | First `preflight doctor` reports zero drift            |
| `capture.completed`   | First `preflight capture` writes a layer file          |

Each event carries: an anonymous machine ID (sha256 of hostname + first-boot
nonce), the event name, and a UTC timestamp. **No paths, package names,
emails, IPs, identifiers, or config contents are ever sent.** The redactor
helpers in `internal/domain/advisor/redact.go` set the bar for what can leave
the machine.

## Consent UX

- Default: **off**. No telemetry without explicit opt-in.
- `preflight init` shows a one-screen prompt: "Preflight is OSS and we don't
  know if it's working for people. Send 4 anonymous activation events?
  [y/N]". Default answer is N.
- Opt-out at any time: `rm ~/.preflight/telemetry.yaml`.
- Setting `PREFLIGHT_TELEMETRY=off` overrides everything.

## Feedback loop

`preflight feedback` opens a GitHub Discussion template prefilled with:

- OS / arch
- Preflight version
- Last 3 events from local `~/.preflight/history`
- Anonymized machine ID (so we can dedupe)

No automatic submission; the user reviews and sends manually.

## Status

- [x] NSM defined (this document)
- [ ] Telemetry event scaffold + opt-in prompt
- [ ] `preflight feedback` command
- [ ] Dashboard reporting weekly TTFSA distribution

The scaffold + feedback command are tracked as separate Roady tasks so the
conceptual decision (this document) lands first.
