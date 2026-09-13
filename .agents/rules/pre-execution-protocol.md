# Rule: Mandatory Pre-Execution Check & English-Only Documentation

## 1. Strict English-Only Documentation Rule
- **ALL documentation, markdown files, guides, execution results, logs, architecture notes, specifications, commit messages, and code comments MUST be written exclusively in English.**
- **No Vietnamese or any other non-English language is permitted anywhere in the repository documentation.**

## 2. Pre-Execution Inspection Protocol
Before executing any terminal/shell command (`run_command`) or modifying any code/configuration files (`replace_file_content`, `write_to_file`), the AI Agent **MUST ALWAYS**:

1. **Briefly review run-context documents**:
   - `run-context/rule.md`: Safety constraints, Go testing protocols, WSL2 execution requirements, and English-only documentation rule.
   - `run-context/context.md`: Active microservices map, database port mappings, and service dependencies.
   - `run-context/results/latest.md`: Latest execution outcome, unresolved issues, and current platform status.

2. **After finishing execution / modifications**:
   - Run verification commands (formatting, tests, builds).
   - Update `run-context/results/latest.md` with the new outcome (in English) and archive a snapshot into `run-context/results/history/`.
