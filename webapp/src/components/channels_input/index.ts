// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import type {Dispatch} from 'redux';

import type {Channel} from '@mattermost/types/channels';

import ChannelTypes from 'mattermost-redux/action_types/channels'
import {getChannel} from 'mattermost-redux/selectors/entities/channels';
import type {ActionFunc, DispatchFunc, GetStateFunc} from 'mattermost-redux/types/actions';
import type {GlobalState} from 'mattermost-redux/types/store';

import Client from '@/client';

import ChannelsInput from './channels_input';

function mapStateToProps(state: GlobalState, ownProps: {channels: Channel[] | Array<{id: string}>}) {
    const channelIds = ownProps.channels ?
        ownProps.channels.map((channel: any) => channel.id) :
        [];

    const channelObjects = channelIds.map((id: string) => {
        const channel = getChannel(state, id);
        return channel || {id};
    });

    return {
        channels: channelObjects,
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            searchChannels,
            getMissingChannelsByIds,
        }, dispatch),
    };
}

// Search channels function - returns a thunk
const searchChannels = (term: string) => {
    return async () => {
        try {
            if (!term) {
                return [];
            }
	    return Client.searchChannels(term);
        } catch (error) {
            console.log('Error searching channels:', error);
	    throw error;
        }
    };
};

// keep track of ongoing requests to ensure we don't try
// to query for the same channels simultaneously
const pendingChannelRequests = new Set<string>()

// Get missing channels by IDs - returns a thunk
export function getMissingChannelsByIds(channelIds: string[]): ActionFunc {
    return async (dispatch: DispatchFunc, getState: GetStateFunc) => {
	const state = getState();
	const {channels} = state.entities.channels
	const missingIds: string[] = [];

	channelIds.forEach((id) => {
	    if (!channels[id] && !pendingChannelRequests.has(id)) {
		missingIds.push(id);
	    }
	});

	if (missingIds.length === 0) {
	    return {data: []};
	}
	    
	missingIds.forEach((id) => pendingChannelRequests.add(id));

	let fetchedChannels = [];

	try {
	    const promises = [];
	    for (const channelId of missingIds) {
		promises.push(Client.getChannel(channelId));
	    }
	    fetchedChannels = await Promise.all(promises);
	} catch (error) {
	    console.log(error);
	    throw error;
	}

	missingIds.forEach((id) => pendingChannelRequests.delete(id));

	if (fetchedChannels.length > 0) {
	    dispatch({
		type: ChannelTypes.RECEIVED_CHANNELS,
		data: fetchedChannels,
	    });
	    return {data: fetchedChannels};
	}
	
	return {data: []};
    };
};

export const ChannelsInputComponent = connect(mapStateToProps, mapDispatchToProps)(ChannelsInput);
