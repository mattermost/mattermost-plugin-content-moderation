// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {getCurrentUser} from 'mattermost-redux/selectors/entities/users';

import {client} from '@/client';
import UserSettings from '@/components/admin_settings/user_settings';
import manifest from '@/manifest';
import type {PluginRegistry} from '@/types/mattermost-webapp';

export default class Plugin {
    private store: any;

    public async initialize(registry: PluginRegistry, store: any) {
        this.store = store;
        registry.registerAdminConsoleCustomSetting('excludedUsers', UserSettings, {showTitle: true});

        registry.registerPostDropdownMenuAction(
            'Enable Channel Moderation',
            this.handleEnableModeration,
        );

        registry.registerPostDropdownMenuAction(
            'Disable Channel Moderation',
            this.handleDisableModeration,
        );
    }

    private manageChannelModeration = async (postId: string, enable: boolean) => {
        try {
            const state = this.store.getState();
            const post = getPost(state, postId);

            if (!post) {
                // eslint-disable-next-line no-console
                console.error('Post not found in store:', postId);
                return;
            }

            const channelId = post.channel_id;

            if (enable) {
                await client.enableChannelModeration(channelId);
            } else {
                await client.disableChannelModeration(channelId);
            }

            const currentUser = getCurrentUser(state);
            if (currentUser) {
                const action = enable ? 'enabled' : 'disabled';
                await client.createEphemeralPost(channelId, `Content moderation has been ${action} for this channel.`, currentUser.id);
            }
        } catch (error) {
            const action = enable ? 'enable' : 'disable';
            // eslint-disable-next-line no-console
            console.error(`Failed to ${action} channel moderation:`, error);

            try {
                const state = this.store.getState();
                const post = getPost(state, postId);
                if (post) {
                    const currentUser = getCurrentUser(state);
                    if (currentUser) {
                        await client.createEphemeralPost(post.channel_id, `Failed to ${action} moderation for this channel.`, currentUser.id);
                    }
                }
            } catch (ephemeralError) {
                // eslint-disable-next-line no-console
                console.error('Failed to create error message:', ephemeralError);
            }
        }
    };

    private handleEnableModeration = async (postId: string) => {
        await this.manageChannelModeration(postId, true);
    };

    private handleDisableModeration = async (postId: string) => {
        await this.manageChannelModeration(postId, false);
    };
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
