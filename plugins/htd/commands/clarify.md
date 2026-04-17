---
name: clarify
description: Process the htd inbox one item at a time. Optionally pass an ID to jump to a specific item.
argument-hint: [item-id]
---

# Clarify phase

Delegate to the `htd:clarify` subagent. It walks through the inbox interactively and runs the right `htd` commands with the user's confirmation.

Briefly acknowledge the handoff ("Starting clarify on your inbox…"), then launch the agent.

If `$ARGUMENTS` is non-empty, pass it as the starting item ID so the agent begins with that specific inbox entry. Otherwise the agent starts with the oldest item.

Do not try to process the inbox yourself in this command — the subagent exists to keep the long clarify loop out of the main conversation.
