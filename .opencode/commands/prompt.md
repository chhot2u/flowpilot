---
description: Optimize, compare, and audit prompts for better LLM results
agent: pro-prompts-enhanced
subtask: false
---

# Prompt Optimization Command

Optimize, compare, or audit the following prompt: $ARGUMENTS

## Your Task

1. **Analyze** the provided prompt across 8 dimensions (clarity, specificity, structure, context, output control, edge cases, chain-of-thought, reusability)
2. **Generate 3 optimized variants** using different strategies (structured, chain-of-thought, maximum precision)
3. **Score all versions** in a comparison matrix (1-10 per dimension)
4. **Recommend the best variant** with clear reasoning

## Modes

If the user specifies a mode, follow it:

- `optimize <prompt>` — Full analysis + 3 variants + comparison + recommendation
- `compare <prompt1> vs <prompt2>` — Side-by-side comparison with scoring
- `audit <prompt>` — Checklist audit with specific improvement suggestions
- `rewrite <prompt> for <model>` — Model-specific optimization
- `simplify <prompt>` — Reduce complexity while preserving effectiveness
- `template <use-case>` — Generate a reusable prompt template

If no mode is specified, default to `optimize`.

## Output Format

```
PROMPT ANALYSIS
===============
[Assessment of original]

OPTIMIZED VARIANTS
==================
[3 variants with full prompt text]

COMPARISON MATRIX
=================
[Scoring table]

RECOMMENDATION
==============
[Best variant + reasoning]
```