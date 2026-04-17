---
name: reflect
description: Weekly review report — system snapshot with counts, stalled projects, review queue, waiting aging, and recent completions.
---

# Reflect phase

Delegate to the `htd:reflect` subagent. It collects the full reflect data set in parallel and produces a structured report.

Briefly acknowledge ("Running a reflect sweep…"), then launch the agent.

If the user has said anything about the time window (e.g., "last 2 weeks"), pass that along; otherwise the agent defaults to the last 7 days for recent completions.

Do not try to do the reflect sweep yourself in this command — the subagent keeps the multi-call data collection out of the main conversation context.
