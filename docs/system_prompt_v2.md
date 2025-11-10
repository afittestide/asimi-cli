
  System Prompt Architecture v2

  The system prompt is build using go templates and is made of the following 

  1. Provider-Specific Core Prompts

  The main system prompt selection depends on the model:

  - Claude models (claude): Uses anthropic.txt
  - GPT models (gpt-, o1, o3): Uses gpt.txt
  - Gemini models (gemini-): Uses gemini.txt
  - Default/fallback: Uses default.txt

  2. Provider Headers

  There's a SystemPrompt.Header() function that returns provider-specific headers:
  - Anthropic providers: returns a "You are Claude Code, Anthropic's official CLI for Claude."
  - Other providers: Get no additional header

  3. Dynamic Environment Context

  Already implemented using the <env>
  4. Custom Instructions

  The SystemPrompt.custom() function loads user-defined instructions from:
  - Local files: AGENTS.md, CLAUDE.md, CONTEXT.md (searched up the directory tree)
  - Global files: ~/.config/opencode/AGENTS.md, ~/.claude/CLAUDE.md
  - Config-defined instruction files: From the instructions array in config

  5. Agent-Specific Prompts

  Individual agents can override the default system prompt via the prompt field in their configuration (see
  packages/opencode/src/agent/agent.ts:99).

  6. Special Purpose Prompts

  There are also specialized prompts for specific functions:
  - summarize.txt: For conversation summarization
  - title.txt: For generating session titles
  - initialize.txt: For session initialization
  - plan.txt: Added when using the "plan" agent mode

  Assembly Process

  The final system prompt is assembled in packages/opencode/src/session/index.ts:725-737:

  1. Header: Provider-specific header (if any)
  2. Core: Either agent-specific prompt OR provider-specific prompt
  3. Environment: Dynamic environment information
  4. Custom: User-defined instructions
  5. Optimization: Combined into max 2 messages for caching efficiency

  Key Differences Between Provider Prompts

  From reading the prompt files:

  - Anthropic (anthropic.txt): Full-featured with TodoWrite tools, concise CLI-style responses
  - Beast (beast.txt): Aggressive autonomous agent mode, heavy web research focus, "never give up" mentality
  - Gemini (gemini.txt): Safety-focused with malicious code detection, structured workflows
  - Qwen (qwen.txt): Similar to Anthropic but without TodoWrite functionality

  This architecture allows the system to provide model-appropriate prompts while maintaining consistency
  through shared environment context and custom instructions.
  ⎿  User rejected Claude's plan:
     ╭────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
     │ I have thoroughly analyzed the system prompt architecture in this coding agent. The research is complete   │
     │ and I've provided a comprehensive explanation of:                                                          │
     │                                                                                                            │
     │ 1. Provider-specific prompts: Different core prompts for Claude, GPT, Gemini, and other models             │
     │ 2. System prompt components: Headers, environment context, custom instructions, and agent overrides        │
     │ 3. Assembly process: How the final prompt is constructed from multiple parts                               │
     │ 4. Key differences: The distinct characteristics of each provider's prompt                                 │
     │                                                                                                            │
     │ No further action is needed - this was a research task that has been fully completed.                      │
     ╰────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
