# MCP_ROLE.md

## What this MCP does (beginner-friendly)

This project ships a small, local MCP (Model Connector Plugin) server that sits
between conversational AI clients (for example, Claude or a G-AI client) and
LabVIEW. The MCP implements the transport and a short, explicit tool set so an
AI can create, wire, save, and run LabVIEW VIs without human intervention.

In plain terms: the AI speaks MCP to a program running on your PC (bridge.exe).
That program turns the AI's high-level commands into safe, validated VI
scripting calls that are forwarded over TCP to a tiny LabVIEW VI (lv_bridge.vi)
which performs the real work inside the LabVIEW IDE. An optional web dashboard
lets you drive the same tools from a phone or browser for testing.

## Why this MCP is different

- Local-first and offline. Many MCP implementations rely on cloud services or
  external tooling. This bridge runs entirely on the local machine and keeps all
  communication on loopback or the LAN — no telemetry, no CDN, and no cloud
  dependencies. That reduces latency and preserves privacy.

- Deterministic, allowlisted tools. Rather than giving an AI general-purpose
  scripting access, the bridge exposes a short, deterministic set of tools
  (create_vi, create_vi_control, add_block_node, wire_nodes, save_vi, run_vi,
  etc.). Every argument is validated at the boundary. That makes behaviour
  predictable and easier to test compared with MCPs that expose broad scripting
  APIs.

- Minimal LabVIEW-side logic. The design intentionally keeps complexity out of
  LabVIEW. The Go bridge computes object ids, grid positions and validation so
  the LabVIEW VI only needs to parse and dispatch commands. That simplifies the
  LabVIEW implementation and reduces the amount of binary work you must maintain
  in the IDE.

- Ephemeral UI sessions. The optional dashboard keeps state in memory only;
  closing the tab destroys the session. No cookies, no localStorage. This is
  deliberate to avoid stale state between AI requests and to make the system
  easier to reason about during development.

- Dual compatibility goal. The bridge supports MCP over stdio so it can be
  registered directly with AI desktop apps (Claude Desktop / Claude Code) and
  also talks to a LabVIEW receiver over TCP. This lets you use the same bridge
  for headless AI workflows and interactive phone-driven inspection.

## What happens if the MCP is absent

If this MCP is not present or not registered with your AI client:

- The AI cannot directly control LabVIEW. Without an MCP the AI has no
  deterministic channel to issue creation and wiring commands into the LabVIEW
  IDE. That means automated VI generation and orchestration stops at the AI
  itself — you would need to perform the steps manually inside LabVIEW.

- Workarounds are manual and error-prone. You can still instruct a human to
  follow a recipe, or use generic VI-scripting tools in LabVIEW, but those
  routes reintroduce manual steps, possible mistakes, and lose the closed-loop
  reproducibility that an MCP provides.

- Testing and automation break. Continuous-generation workflows (for example,
  programmatically creating and running test VIs on demand) rely on the MCP's
  deterministic toolset. Without it you lose that automation capability.

## How this bridge connects G-AI and Claude to VIs (concrete example)

1. Register the bridge as an MCP server with your AI client (Claude or another
   client that supports MCP). The AI then calls the bridge using the familiar
   MCP transport.
2. The AI requests: `create_vi` → `create_vi_control knob` → `create_vi_control
   knob` → `add_block_node add` → `wire_nodes knob1 add1` → `wire_nodes add1
   indicator1` → `save_vi C:\\vis\\adder.vi` → `run_vi`.
3. The bridge validates each argument, assigns ids and grid positions, builds
   the protocol payload and forwards commands to `lv_bridge.vi` over TCP.
4. `lv_bridge.vi` applies the requested changes using VI Scripting inside the
   LabVIEW IDE and returns success/failure responses.

This clear separation — AI intent → validated bridge → thin LabVIEW executor —
keeps the system robust and auditable.

## Security and reliability notes

- Input sanitization is critical: the bridge strips dangerous characters and
  allowlists control types. The TCP command format is deliberately simple so the
  bridge can reliably parse and reject malformed or unexpected input.

- Keep your LabVIEW development environment isolated. Because VI Scripting
  requires the IDE, run the bridge on trusted machines only.

- Use `-mock` during development. The built-in mock LabVIEW daemon lets you
  exercise the full AI → bridge → dashboard chain without running LabVIEW.

## When to use this MCP vs other approaches

- Use this bridge when you need reproducible, local-first VI generation with
  low-latency and strong input validation. It's a good fit for development,
  teaching, and sensitive environments where cloud services are unacceptable.

- If you need deep, exploratory editing of existing VIs or visual diffing of
  complex diagrams, a more feature-rich LabVIEW plugin (or manual LabVIEW
  scripting) may be appropriate; this bridge focuses on creation and simple
  inspection tools rather than being a full IDE replacement.

- Prefer this MCP for automation pipelines: its deterministic API, ephemeral
  sessions, and minimal LabVIEW surface make it easier to integrate into test
  rigs and CI workflows that exercise generated VIs.

## Closing note

This bridge is intentionally opinionated: small, local, and predictable. That
opinion simplifies both AI-side reasoning (the AI has a small, testable toolset)
and the LabVIEW-side implementation (a tiny, reliable executor). If you want a
custom workflow or additional tools, open an issue or a pull request — the
codebase is compact and designed to be extended safely.
