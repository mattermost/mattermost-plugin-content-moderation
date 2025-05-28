// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {connect} from 'react-redux';

import type {Channel} from '@mattermost/types/channels';

import {ChannelsInputComponent} from '../channels_input';

interface ChannelSettingsProps {
    id: string;
    value?: string;
    onChange: (id: string, value: string) => void;
}

interface ChannelSettingsState {
    channels: Array<{id: string}>;
}

class ChannelSettings extends React.Component<ChannelSettingsProps, ChannelSettingsState> {
    constructor(props: ChannelSettingsProps) {
        super(props);

	this.state = {
	    channels: [],
	};
    }

    componentDidMount() {
	this.initializeChannels(this.props.value || '');
    }

    componentDidUpdate(prevProps: ChannelSettingsProps) {
	if (prevProps.value !== this.props.value) {
	    this.initializeChannels(this.props.value || '');
	}
    }

    initializeChannels = (value: string) => {
        if (value) {
            const channelIds = value.split(',').map((id) => id.trim()).filter((id) => id);
            if (channelIds.length > 0) {
                // Just create channel objects with IDs
                // The ChannelsInput component will fetch the full channel details
                const channelObjects = channelIds.map((id) => ({id}));
                this.setState({channels: channelObjects});
            } else {
                this.setState({channels: []});
            }
        } else {
            this.setState({channels: []});
        }
    };

    handleChange = (channels: Channel[]) => {
	if (!channels || !this.props.onChange || !this.props.id) {
	    return;
	}
	const channelIds = channels.map((channel) => channel?.id).filter(Boolean).join(',');
	this.props.onChange(this.props.id, channelIds);
    };

    render() {
	if (!this.props.id) {
	    return null;
	}
        return (
            <ChannelsInputComponent
p                placeholder='Search for channels to exclude from moderation'
                channels={this.state.channels}
                onChange={this.handleChange}
            />
        );
    }
}

export default connect(null, null)(ChannelSettings);
