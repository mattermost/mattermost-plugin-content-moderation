# Mattermost Content Moderator Plugin

[![Build Status](https://github.com/mattermost/mattermost-plugin-content-moderator/actions/workflows/ci.yml/badge.svg)](https://github.com/mattermost/mattermost-plugin-content-moderator/actions/workflows/ci.yml)

This plugin provides content moderation capabilities for Mattermost using Azure AI Content Safety APIs.

## Overview

The Content Moderator Plugin allows Mattermost administrators to ensure all content shared on the platform meets community guidelines by automatically moderating messages and attachments.

Key features:
- Text content moderation (hate speech, sexual content, violence, self-harm)
- Configuration with a single moderation threshold
- Support for targeting specific users (e.g., AI bots) or all users
- Detailed audit logging of moderation actions

## Installation

1. Download the latest release from the [releases page](https://github.com/mattermost/mattermost-plugin-content-moderator/releases)
2. Upload the plugin to your Mattermost instance via System Console > Plugin Management
3. Enable the plugin
4. Configure the plugin with your Azure AI Content Safety API key and settings

## Configuration

Configuration options:

| Setting | Description |
|---------|-------------|
| Enabled | Enable/disable content moderation |
| Type | Moderation provider type (currently only "azure" is supported) |
| Endpoint | API endpoint |
| API Key | API key (kept secure) |
| Moderate All Users | When enabled, content from all users will be moderated |
| Moderation Targets | If not moderating all users, the plugin moderates content from these User IDs |
| Threshold | Single severity threshold applied to all content categories |

The Azure AI Content Safety API uses severity levels from 0-6:
- 0: Safe (always allowed)
- 2: Low severity (mild)
- 4: Medium severity (moderate)
- 6: High severity (severe)

## How It Works

The plugin intercepts messages before they are posted to the database using Mattermost's server-side hooks:

1. When a user creates or updates a message, the plugin checks if that user should be moderated.
2. If moderation applies, the content is sent to Azure AI Content Safety for analysis.
3. If any content category exceeds the configured threshold, the message is blocked and the user receives feedback.
4. If content passes moderation checks, the message is allowed through normally.

The plugin uses a "fail-closed" approach - if the moderation service encounters an error, content is blocked by default for safety.

## AI Plugin Integration

This plugin is designed to work with the Mattermost AI Plugin, particularly for moderating AI-generated responses. For proper integration:

1. Install both the Content Moderator Plugin and the AI Plugin
2. Configure the AI Plugin to disable streaming responses
3. Add the AI bot user ID to the moderation targets list in the Content Moderator Plugin

## Development

### Prerequisites

- Go 1.22+
- Node.js v16+ and NPM v8+
- Make

### Building the Plugin

```bash
make
```

### Deploying with Local Mode

If your Mattermost server is running locally, you can enable [local mode](https://docs.mattermost.com/administration/mmctl-cli-tool.html#local-mode) to streamline deploying your plugin:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_LOCALSOCKETPATH=/var/tmp/mattermost_local.socket
make deploy
```

## License

This repository is licensed under the [Mattermost Source Available License](LICENSE) license.
