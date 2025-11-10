# Init Command Flow

## Overview

The init command analyzes a codebase and generates an AGENTS.md file containing build commands, test commands, and code style guidelines. This file becomes part of the system prompt for all future AI sessions in the project.

## Trigger Points

The init command can be triggered in two ways:

- Via REST API endpoint: `POST /session/:id/init`
- Via keyboard shortcut: Default `<leader>i` (configured as `project_init`)

## Session Initialization Process

### Entry Point

`Session.initialize()` function receives:

- `sessionID`: The session identifier
- `modelID`: AI model to use
- `providerID`: Provider for the AI model
- `messageID`: Unique message identifier

## Core Execution Steps

### 1. Create Initialization Prompt

- Loads the initialization prompt template from `initialize.txt`
- The prompt instructs the AI to analyze the codebase and create an `AGENTS.md` file
- The prompt template is customized with the current worktree path
- Template content requests:
  - Analysis of build/lint/test commands
  - Code style guidelines (imports, formatting, types, naming, error handling)
  - Concise output (about 20 lines)
  - Check for existing Cursor rules or Copilot instructions
  - If an AGENTS.md already exists, improve it

### 2. Submit to Session Prompt System

Calls `SessionPrompt.prompt()` with:

- Session ID
- Message ID
- Model configuration (provider + model)
- Text part containing the initialization prompt

## Session Prompt Processing

### 3. Message Creation

- Creates a user message with the initialization prompt
- Touches the session (updates last activity timestamp)
- Checks if session is busy; if so, queues the request

### 4. Agent & Model Resolution

- Resolves the agent (defaults to "build" agent if not specified)
- Gets the specified model from the provider
- Validates model availability and configuration

### 5. System Prompt Assembly

Builds comprehensive system prompt by concatenating:

**Provider-specific header** (e.g., Anthropic spoofing if needed)

**Base provider prompt** (different for GPT, Gemini, Claude, etc.)

**Environment information:**

- Current working directory
- Git repository status
- Platform (OS)
- Current date

**Project structure** (up to 200 files from git tree)

**Custom instructions** from:

- Local AGENTS.md (searches up from current directory)
- Local CLAUDE.md
- Local CONTEXT.md (deprecated)
- Global AGENTS.md in config directory
- Global CLAUDE.md in ~/.claude/
- Additional paths from config instructions

### 6. Tool Resolution

- Assembles available tools for the agent
- Configures tool permissions and capabilities
- Tools include: Read, Write, Edit, Bash, List, Glob, Grep, Task, etc.

### 7. Message History Retrieval

- Fetches all messages in the session
- Filters out summarized messages
- Checks for token overflow and compacts if needed
- Inserts reminder messages based on agent configuration

### 8. AI Stream Processing

- Initiates streaming text generation with the AI model
- Processes the stream incrementally:
  - Text chunks are captured
  - Tool calls are detected and executed
  - Results are fed back to the model
  - Process continues until completion

### 9. Tool Execution Loop

When the AI decides to use tools:

- Tool calls are extracted from the stream
- Each tool is invoked with provided parameters
- Tool results are stored as new message parts
- Control returns to the AI with tool results
- AI continues with next step or completes

## Project Initialization Marking

After prompt completes:

- Calls `Project.setInitialized(projectID)`
- Updates project storage with: `time.initialized = Date.now()`
- This marks the project as having been analyzed

## Project Context Discovery

### Project Identification

- Searches up the directory tree for `.git` folder
- If found:
  - Gets git root directory
  - Runs `git rev-list --max-parents=0 --all` to get first commit SHA
  - Uses first commit as project ID (unique identifier)
  - Gets absolute path to git toplevel
- If not found:
  - Creates "global" project with "/" as worktree

### Project Storage

Creates/updates project entry in storage:

- `id`: Git first commit SHA or "global"
- `worktree`: Git root or "/"
- `vcs`: "git" if in git repo
- `time.created`: Current timestamp
- `time.initialized`: Set after init completes

## File Analysis & AGENTS.md Generation

The AI agent:

- Uses Read/List/Glob tools to explore the codebase
- Identifies key files (package.json, build configs, test files)
- Analyzes code patterns and conventions
- Checks for existing documentation
- Generates or updates AGENTS.md with:
  - Build commands
  - Test commands (including single test execution)
  - Linting commands
  - Code style guidelines
  - Import conventions
  - Naming conventions
  - Error handling patterns
  - Any project-specific rules

## Completion & Storage

- The generated AGENTS.md is written to the project root
- Session state is updated
- Project is marked as initialized
- Future sessions will automatically include this AGENTS.md in their system prompt

## Subsequent Usage

Once initialized:

- All future AI sessions in this project will load AGENTS.md
- The file is included in the system prompt automatically
- Agents have immediate context about project conventions
- No need to re-explain build/test/style guidelines

## Key Design Principles

1. **One-time Analysis**: Init is designed to run once per project, creating persistent guidance
2. **Hierarchical Rules**: Supports both local (per-project) and global (user-wide) instruction files
3. **Automatic Discovery**: Searches up directory tree for context files
4. **Git-Based Identity**: Uses git history to uniquely identify projects
5. **Streaming AI**: Uses streaming responses for real-time feedback
6. **Tool Augmentation**: AI can use filesystem tools to analyze codebase structure
7. **Caching Context**: Generated AGENTS.md becomes part of system prompt cache

## Data Flow Diagram

```
User Trigger (API/Keyboard)
    ↓
Session.initialize()
    ↓
SessionPrompt.prompt()
    ↓
├─ Create User Message
├─ Resolve Agent & Model
├─ Build System Prompt
│   ├─ Provider Header
│   ├─ Base Prompt
│   ├─ Environment Info
│   ├─ Project Tree
│   └─ Custom Instructions (AGENTS.md, etc.)
├─ Resolve Tools
├─ Get Message History
└─ Stream AI Response
    ↓
AI Tool Execution Loop
    ├─ Read files
    ├─ List directories
    ├─ Analyze patterns
    └─ Write AGENTS.md
    ↓
Project.setInitialized()
    ↓
Complete
```
