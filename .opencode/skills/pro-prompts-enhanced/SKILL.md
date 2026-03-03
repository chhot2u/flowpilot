---
name: pro-prompts-enhanced
description: Prompt optimization and comparison specialist. Analyzes prompts, generates optimized variants, scores them side-by-side, and recommends the best version for any LLM. Use when crafting, improving, or comparing prompts for better AI results.
---

# Pro Prompts Enhanced

Optimize any prompt through structured analysis, variant generation, side-by-side comparison, and evidence-based recommendation.

## When to Activate

- Writing a new prompt and wanting the best possible version
- Comparing two or more prompt approaches to pick a winner
- Debugging a prompt that produces inconsistent or poor results
- Adapting a prompt from one model to another (Claude, GPT, Gemini, open models)
- Building reusable prompt templates for recurring tasks
- Reviewing prompt quality before deploying to production
- Teaching prompt engineering principles through hands-on examples

## Core Workflow: Compare-First

Every optimization follows this sequence:

### 1. Analyze

Evaluate the original prompt across 8 dimensions:

| Dimension | What to Check |
|-----------|---------------|
| Clarity | Is the intent unambiguous? One clear task? |
| Specificity | Are constraints, scope, and boundaries defined? |
| Structure | Does it use sections, delimiters, role framing? |
| Context | Enough background for the model to succeed? |
| Output Control | Format, length, tone, audience specified? |
| Edge Cases | Handles ambiguity, missing data, failure modes? |
| Chain-of-Thought | Encourages reasoning before answering? |
| Reusability | Works across inputs or is it one-shot only? |

### 2. Generate 3 Variants

**Variant A — Structured Enhancement**
Role assignment, explicit constraints, output format, delimiters.

**Variant B — Chain-of-Thought / Reasoning**
Step-by-step instructions, few-shot examples, self-verification.

**Variant C — Maximum Precision**
Exact output schema, negative constraints, edge case handling, evaluation criteria.

### 3. Score with Comparison Matrix

```
| Dimension        | Original | Variant A | Variant B | Variant C |
|-----------------|----------|-----------|-----------|-----------|
| Clarity          |   /10    |    /10    |    /10    |    /10    |
| Specificity      |   /10    |    /10    |    /10    |    /10    |
| Structure        |   /10    |    /10    |    /10    |    /10    |
| Context          |   /10    |    /10    |    /10    |    /10    |
| Output Control   |   /10    |    /10    |    /10    |    /10    |
| Edge Cases       |   /10    |    /10    |    /10    |    /10    |
| Chain-of-Thought |   /10    |    /10    |    /10    |    /10    |
| Reusability      |   /10    |    /10    |    /10    |    /10    |
| TOTAL            |   /80    |    /80    |    /80    |    /80    |
```

### 4. Recommend

- Which variant wins and why
- What specific techniques drove the improvement
- When to use each variant (model, context, audience)

## Prompt Engineering Techniques

### Role Priming
```
You are a [specific expert] with [years] of experience in [domain].
Your task is to [action] for [audience].
```

### Structured Output
```
Respond in this exact format:
## Analysis
[content]
## Recommendation
[content]
## Confidence
[HIGH/MEDIUM/LOW]
```

### Constraint Specification
```
Rules:
- Maximum 200 words
- Use bullet points
- Include exactly 3 examples
- Do NOT include [exclusion]
- If uncertain, say "I'm not sure"
```

### Few-Shot Examples
```
Input: [example 1]
Output: [example 1]

Input: [example 2]
Output: [example 2]

Now process:
Input: [actual input]
```

### Chain-of-Thought
```
Think step by step:
1. Identify [aspect 1]
2. Analyze [aspect 2]
3. Consider [aspect 3]
4. Synthesize your answer

Show reasoning before your final answer.
```

### Self-Verification
```
After generating your response:
1. Check all constraints are met
2. Verify output format matches spec
3. Confirm no hallucinated information
4. Rate confidence: HIGH/MEDIUM/LOW
```

### Negative Prompting
```
Do NOT:
- Make up facts or statistics
- Use jargon without explanation
- Exceed the specified length
- Repeat the question
- Give generic advice
```

### Context Windowing
```
Background: [relevant context]
Current Situation: [specific scenario]
Task: [what to do]
Constraints: [boundaries]
```

### Meta-Prompting
```
Before answering, plan your approach:
- What information do you need?
- What structure best serves the user?
- What misunderstandings should you prevent?
Then execute your plan.
```

## Prompt Audit Checklist

- [ ] Clear role or persona assignment
- [ ] Task stated in one unambiguous sentence
- [ ] Output format specified (JSON, markdown, bullets, prose)
- [ ] Length constraints defined
- [ ] Relevant context provided
- [ ] Delimiters used for multi-part inputs
- [ ] Audience and tone specified
- [ ] Edge cases handled
- [ ] Examples included when format matters
- [ ] Negative constraints present
- [ ] Reasoning requested before conclusions
- [ ] Self-check or verification step included

## Model-Specific Optimization

### Claude (Anthropic)
- XML tags for structure: `<context>`, `<task>`, `<format>`
- System prompts for persistent instructions
- "Think step by step" for complex reasoning
- Important instructions at beginning AND end

### GPT (OpenAI)
- System/user message separation
- JSON mode for structured output
- Function calling for tool-use
- Explicit temperature guidance

### Gemini (Google)
- Multimodal capabilities when relevant
- Clear section headers
- Grounding context for factual accuracy

### Open Models (Llama, Mistral)
- Simpler, more explicit prompts
- Fewer implicit assumptions
- More examples for complex formats
- Concise for shorter context windows

## Anti-Patterns to Fix

| Bad Pattern | Fix |
|-------------|-----|
| "Make it good" | "Achieve 95%+ accuracy on edge cases" |
| "Just answer" | "Respond in bullet points, max 5 items" |
| "Summarize this" | "Summarize for a technical audience in 3 sentences" |
| "Write about X" | "Write 500 words about X, focusing on Y, avoiding Z" |
| "Fix the bug" | "Fix the null pointer exception in auth.ts:42" |
| "What's the answer" | "Explain your reasoning, then give the answer" |
| No examples | Provide 2-3 concrete input/output examples |
| No error handling | "If data is malformed, return error with field name" |

## Quick Modes

| Command | Action |
|---------|--------|
| `/prompt optimize <text>` | Full analysis + 3 variants + matrix + recommendation |
| `/prompt compare <A> vs <B>` | Side-by-side scoring of two prompts |
| `/prompt audit <text>` | Checklist audit with improvement suggestions |
| `/prompt rewrite <text> for <model>` | Model-specific optimization |
| `/prompt simplify <text>` | Reduce complexity, preserve effectiveness |
| `/prompt template <use-case>` | Generate reusable prompt template |

## Output Format

Every response follows this structure:

```
PROMPT ANALYSIS
===============
Original: [assessment]
Key Issues: [numbered list]

OPTIMIZED VARIANTS
==================
Variant A: [full prompt text]
Variant B: [full prompt text]
Variant C: [full prompt text]

COMPARISON MATRIX
=================
[scoring table]

RECOMMENDATION
==============
Best: [Variant X]
Why: [reasoning]
Alternatives: [when to use others]

APPLIED TECHNIQUES
==================
[techniques used and why]
```