// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface PostDropdownMenuActionProps {
    postId: string;
    channelId: string;
    teamId: string;
    userId: string;
}

export interface PluginRegistry {
    registerAdminConsoleCustomSetting(key: string, component: React.ComponentType<any>, options?: {showTitle: boolean});
    registerPostDropdownMenuAction(
        text: string | React.ReactElement,
        action: (postId: string) => void,
        filter?: (postId: string) => boolean
    ): void;
    registerChannelHeaderMenuAction(
        text: string,
        action: (channelId: string) => void
    ): void;
}

export interface PluginManifest {
    id: string;
    version: string;
}
