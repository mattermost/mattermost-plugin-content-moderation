# Mattermost Content Moderation Plugin

This plugin provides content moderation capabilities for Mattermost using Azure AI Content Safety APIs.

This plugin requires an active enterprise license of Mattermost.

## Overview

The Content Moderation Plugin allows Mattermost administrators to ensure all content shared on the platform meets community guidelines by automatically moderating messages and attachments.

Key features:
- Text content moderation (hate speech, sexual content, violence, self-harm)
- Configuration with a single moderation threshold
- Moderation of all users with ability to exclude specific users

## Installation

1. Download the latest release from the [releases page](https://github.com/mattermost/mattermost-plugin-content-moderation/releases)
2. Upload the plugin to your Mattermost instance via System Console > Plugin Management
3. Enable the plugin
4. Configure the plugin with your Azure AI Content Safety API key and settings

## Configuration

Configuration options:

| Setting | Description |
|---------|-------------|
| Enabled | Enable/disable content moderation |
| Type | Moderation provider type (currently only "azure" is supported) |
| Azure Endpoint | Azure API endpoint |
| Azure API Key | Azure API key (kept secure) |
| Exclude Direct/Group Messages | When enabled, direct messages and group messages will not be moderated |
| Exclude Private Channels | When enabled, private channels will not be moderated |
| Excluded Users | User IDs to exclude from content moderation. All other users will be moderated |
| Excluded Channels | Channel IDs to exclude from content moderation. Messages in these channels will not be moderated |
| Bot Username | The username displayed for moderation notifications |
| Azure Threshold | Single severity threshold applied to all content categories |

The Azure AI Content Safety API uses severity levels from 0-6:
- 0: Safe (always allowed)
- 2: Low severity (mild)
- 4: Medium severity (moderate)
- 6: High severity (severe)

## License

This repository is licensed under the [Mattermost Source Available License](LICENSE) license.

## Frequently Asked Questions

### How does content moderation work?

When a user posts a message, it appears immediately in the channel. The plugin then analyzes the content in the background using Azure AI Content Safety APIs. If harmful content is detected, the post is automatically deleted and notifications are sent to inform users of the removal.

### Will I still receive notifications for harmful content?

Currently, yes. Push notifications may be sent for posts that contain harmful content before the moderation process completes. This is because notifications are typically sent immediately when posts are created, while content analysis happens asynchronously. We are working to improve this behavior (see roadmap).

### Can I exclude certain users from moderation?

Yes, you can specify user IDs in the "Excluded Users" configuration setting. All other users will have their content moderated automatically.

### Can I exclude certain channels from moderation?

Yes, you have several options for excluding channels from moderation:

1. **Channel Type Exclusions**: Use the "Exclude Direct/Group Messages" option to disable moderation for all direct messages and group messages. Use the "Exclude Private Channels" option to disable moderation for all private channels.

2. **Specific Channel Exclusions**: Specify individual channel IDs in the "Excluded Channels" configuration setting. Messages in these specific channels will not be moderated, regardless of the user who posted them.

### What if content moderation APIs are unavailable?

The plugin uses a "fail-open" approach for reliability. If the moderation API is unavailable or returns an error, no posts are moderated. When this occurs, you'll see error messages in the server logs like:

```
Content moderation error err="moderation service is not available" post_id="abc123" user_id="xyz789"
```

### How can I monitor moderation activity?

Moderation activity is logged in the Mattermost server logs. When content is flagged and removed, you'll see log entries like:

```
Content was flagged by moderation post_id="abc123" severity_threshold=2 computed_severity_hate=4 computed_severity_violence=3
```

This shows which post was flagged, the configured threshold, and the computed severity scores for each category that exceeded the threshold. Future versions will include metrics visualization support for better monitoring and reporting.

## Roadmap

- [ ] Implement notification blocking for posts under moderation
- [X] Support writing moderation events to the Audit log
- [ ] Add local LLM option as the moderator backend
- [ ] Support excluding users from moderation by group
- [ ] Support moderating text attachments
- [ ] Support moderating images
- [ ] Add metrics visualization support (Grafana)
