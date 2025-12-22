Preflight — Vision & Principles
Vision
Preflight enables anyone to compile a reproducible, explainable, and portable workstation.
A workstation should be deterministic, transparent, safe to change, easy to recreate, and fully owned by the user. Preflight treats workstation setup as a compilation problem, not a collection of scripts and ad‑hoc installers.

---

What Preflight Is
Preflight is:
• A deterministic workstation compiler
• With an optional guided discovery layer
• Producing plain, human‑owned configuration
• Converging machines via plan → apply → verify
Preflight works for:
• Engineers and non‑engineers
• Personal and work machines
• Online or fully offline environments

---

What Preflight Is Not
Preflight is not:
• A SaaS‑first product
• An MDM or remote control system
• A background agent that mutates your machine
• A replacement for your creative dotfiles workflows
• A Nix replacement (Preflight borrows ideas, not ideology)

---

Core Guarantees
Preflight always guarantees:

1. No execution without a plan
2. Every change is explainable
3. Re-running is safe and idempotent
4. Configuration is portable and inspectable
5. Secrets never leave the machine
6. AI never executes actions
7. User ownership over all outputs

---

Compiler Model
Preflight operates like a compiler:
Intent (layers, profiles, capabilities)
↓
Merge & normalize
↓
Plan (diff + explanation)
↓
Apply (deterministic)
↓
Verify (doctor / drift)
Execution determinism is controlled via modes:
• intent — install latest compatible
• locked — prefer lockfile; update intentionally
• frozen — fail on lock mismatches

---

AI Philosophy
AI in Preflight is:
• Optional
• BYOK (bring your own key / provider)
• Advisory only
• Works with cloud providers or local models (e.g., Ollama)
AI may:
• Guide onboarding interviews
• Suggest tools, presets, and capability packs
• Explain why something is selected (with tradeoffs)
• Infer profiles/layers from an existing machine
• Link to relevant docs and provide quick demos/tours
AI may never:
• Execute commands
• Mutate the system
• Access secrets
• Override user approval

---

Dotfiles Philosophy
Dotfiles are first‑class artifacts produced and managed by the compiler.
Preflight supports three dotfile modes:

1. Generated — Preflight owns the file (best for beginners/non‑engineers)
2. Template + user overrides — Preflight manages a base; users extend safely
3. Bring-your-own — Preflight links/validates; never rewrites
   Preflight never silently overwrites user changes. Dotfile diffs appear in preflight plan, and drift is detected by preflight doctor.

---

Long-Term Direction
Preflight starts as a compiler.
Over time it may grow:
• A plugin/capability marketplace
• Curated presets and packs for different personas
• Org baselines (still local-first)
• Richer discovery guidance and learning tours
But it will always remain:
• Local-first
• Transparent and explainable
• Deterministic
• Git-native
• BYOK for AI
Chat
🏗️
Preflight
4 sources
Preflight is a local-first, terminal-based tool designed to transform workstation setup into a deterministic compilation process. By using a declarative configuration model, it allows users to plan, apply, and verify reproducible environments through a series of structured layers and modules. The system prioritizes transparency and safety, offering human-readable plans and an interactive "doctor" mode to detect configuration drift. Users can bootstrap new setups via AI-guided discovery or reverse-engineer existing machines into portable, git-friendly configurations. While it supports automated dotfile management and package installation, it remains a trust-first tool where AI acts only as an advisor and never executes code directly. Ultimately, Preflight ensures that personal and professional environments are portable, inspectable, and fully user-owned.

How does Preflight use a compiler model to ensure deterministic workstation setup?
In what ways does Preflight maintain user trust through explainability and safety?
How do layers and reproducibility modes facilitate portable and managed workstation configurations?
Start typing...
4 sources
Studio
Audio Overview
Video Overview
Mind Map
Reports
Flashcards
Quiz
Infographic
Slide Deck
Data Table
