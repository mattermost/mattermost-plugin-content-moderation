// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from './manifest';

export interface ModerationStatusResponse {
    enabled: boolean;
    excluded: boolean;
}

export class Client {
    private baseUrl: string;

    constructor() {
        this.baseUrl = `/plugins/${manifest.id}`;
    }

    async enableChannelModeration(channelId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/channels/${channelId}/moderation/enable`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            credentials: 'same-origin',
        });

        if (!response.ok) {
            throw new Error(`Failed to enable channel moderation: ${response.statusText}`);
        }
    }

    async disableChannelModeration(channelId: string): Promise<void> {
        const response = await fetch(`${this.baseUrl}/channels/${channelId}/moderation/disable`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            credentials: 'same-origin',
        });

        if (!response.ok) {
            throw new Error(`Failed to disable channel moderation: ${response.statusText}`);
        }
    }

    async getChannelModerationStatus(channelId: string): Promise<ModerationStatusResponse> {
        const response = await fetch(`${this.baseUrl}/channels/${channelId}/moderation/status`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            credentials: 'same-origin',
        });

        if (!response.ok) {
            throw new Error(`Failed to get channel moderation status: ${response.statusText}`);
        }

        return response.json();
    }

    async createEphemeralPost(channelId: string, message: string, userId: string): Promise<void> {
        const response = await fetch('/api/v4/posts/ephemeral', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            credentials: 'same-origin',
            body: JSON.stringify({
                user_id: userId,
                post: {
                    channel_id: channelId,
                    message,
                },
            }),
        });

        if (!response.ok) {
            throw new Error(`Failed to create ephemeral post: ${response.statusText}`);
        }
    }
}

export const client = new Client();
