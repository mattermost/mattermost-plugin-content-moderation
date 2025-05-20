# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**IMPORTANT**: This CLAUDE.md file MUST be kept up to date whenever code changes are made, with NO EXCEPTIONS. Any changes to the codebase should be reflected in this document.

## Project Context
This repository contains a Mattermost Content Moderation plugin that provides automatic content moderation using Azure AI Content Safety APIs. Key features:

- Text content moderation (hate speech, sexual, violence, self-harm)
- Configurable single moderation threshold
- User targeting (specific users or all users)
- Plugin hooks for message posting and editing
- Fail-closed approach for API failures
- Integration with Mattermost AI Plugin

The core components include:
- `moderation/moderator.go`: Core moderation interface
- `moderation/azure/azure.go`: Azure AI Content Safety implementation
- `plugin.go`: Main plugin with hooks for message moderation
- `configuration.go`: Plugin settings management

## Build Commands
- `make all`: Run check-style, test, and build the plugin
- `make check-style`: Run linters for server (golangci-lint) and webapp (eslint)
- `make test`: Run all tests
- `make dist`: Build and bundle the plugin
- `make deploy`: Build and install the plugin to a local Mattermost server

## Code Style Guidelines
- **Imports**: Standard Go import organization (stdlib, external, internal)
- **Formatting**: Use `go fmt` for Go code and ESLint for JavaScript/TypeScript
- **Types**: Prefer explicit types; use interfaces for mocking
- **Error Handling**: Use wrapped errors with context (`errors.Wrap`)
- **Naming**: CamelCase for exported functions, lowerCamelCase for unexported
- **Logging**: Use structured logging via `p.API.LogInfo/LogError` with key-value pairs
- **Plugin IDs**: Must match between package names and plugin.json
- **Config**: Changes to configuration must update both plugin.json and configuration.go

## Plugin Architecture
- Follow Mattermost plugin patterns with server/ and webapp/ directories
- Use hooks defined in plugin.go for server-side integration
- Maintain separation of concerns with modular organization
- Plugin hooks for message interception (MessageWillBePosted, MessageWillBeUpdated)
- Single moderator interface with Azure implementation
- Configuration with a single threshold value instead of per-category thresholds
