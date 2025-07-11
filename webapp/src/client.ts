// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Client4, ClientError} from '@mattermost/client';

import manifest from './manifest';

export interface ModerationStatusResponse {
    enabled: boolean;
    excluded: boolean;
}

export class Client {
    private baseUrl: string;
    private client4: Client4;

    constructor() {
        this.baseUrl = `/plugins/${manifest.id}`;
        this.client4 = new Client4();
    }

    async enableChannelModeration(channelId: string): Promise<void> {
        const url = `${this.baseUrl}/channels/${channelId}/moderation/enable`;
        const options = {
            method: 'POST',
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (!response.ok) {
            const text = await response.text();
            throw new ClientError(this.client4.url, {
                message: text || 'Failed to enable channel moderation',
                status_code: response.status,
                url,
            });
        }
    }

    async disableChannelModeration(channelId: string): Promise<void> {
        const url = `${this.baseUrl}/channels/${channelId}/moderation/disable`;
        const options = {
            method: 'POST',
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (!response.ok) {
            const text = await response.text();
            throw new ClientError(this.client4.url, {
                message: text || 'Failed to disable channel moderation',
                status_code: response.status,
                url,
            });
        }
    }

    async getChannelModerationStatus(channelId: string): Promise<ModerationStatusResponse> {
        const url = `${this.baseUrl}/channels/${channelId}/moderation/status`;
        const options = {
            method: 'GET',
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (!response.ok) {
            const text = await response.text();
            throw new ClientError(this.client4.url, {
                message: text || 'Failed to get channel moderation status',
                status_code: response.status,
                url,
            });
        }

        return response.json();
    }

    async createEphemeralPost(channelId: string, message: string, userId: string): Promise<void> {
        const url = '/api/v4/posts/ephemeral';
        const options = {
            method: 'POST',
            body: JSON.stringify({
                user_id: userId,
                post: {
                    channel_id: channelId,
                    message,
                },
            }),
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (!response.ok) {
            const text = await response.text();
            throw new ClientError(this.client4.url, {
                message: text || 'Failed to create ephemeral post',
                status_code: response.status,
                url,
            });
        }
    }
}

export const client = new Client();
