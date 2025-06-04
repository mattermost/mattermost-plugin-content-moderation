// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import debounce from 'lodash/debounce';
import React, {useEffect} from 'react';
import type {MultiValue, StylesConfig} from 'react-select';
import AsyncSelect from 'react-select/async';

import type {Channel} from '@mattermost/types/channels';

import type {ActionFunc} from 'mattermost-redux/types/actions';

interface ChannelsInputProps {
    placeholder?: string;
    channels: Channel[] | Array<{id: string}>;
    onChange?: (channels: Channel[]) => void;
    actions: {
        searchChannels: (term: string) => Promise<Channel[]>;
        getMissingChannelsByIds: (channelIds: string[]) => ActionFunc;
    };
}

// ChannelsInput searches and selects channels displayed by display name.
// Channels prop can handle the channel object or strings directly if the channel object is not available.
// Returns the selected channel ids in the `OnChange` value parameter.
export default function ChannelsInput(props: ChannelsInputProps) {
    // Extract the channel IDs from the props.channels array
    const channelIds = React.useMemo(() => {
        if (!props.channels || !props.channels.length) {
            return [];
        }
        return props.channels.map((channel: Channel | {id: string}) => {
            return channel.id;
        }).filter(Boolean);
    }, [props.channels]);

    // Fetch missing channels whenever channelIds changes
    useEffect(() => {
        if (channelIds.length > 0) {
            props.actions.getMissingChannelsByIds(channelIds);
        }
    }, [channelIds, props.actions]);

    const onChange = (newValue: MultiValue<string | Channel | {id: string}>) => {
        if (props.onChange) {
            props.onChange(newValue as unknown as Channel[]);
        }
    };

    const getOptionValue = (channel: Channel | {id: string} | string) => {
        if (typeof channel === 'object' && channel.id) {
            return channel.id;
        }
        return channel as string;
    };

    const formatOptionLabel = (option: Channel | {id: string} | string) => {
        if (typeof option === 'object') {
            if ('display_name' in option && option.display_name) {
                return (
                    <React.Fragment key={option.id}>
                        {option.display_name}
                    </React.Fragment>
                );
            }

            if ('name' in option && option.name) {
                return (
                    <React.Fragment key={option.id}>
                        {option.name}
                    </React.Fragment>
                );
            }

            if (option.id) {
                return option.id;
            }
        }

        return option as string;
    };

    const debouncedSearchChannels = debounce((term: string, callback: (data: Channel[]) => void) => {
        props.actions.searchChannels(term).
            then((data) => {
                callback(data);
            }).
            catch(() => {
                // eslint-disable-next-line no-console
                console.error('Error searching channels in custom attribute settings dropdown.');
                callback([]);
            });
    }, 150);

    const channelsLoader = (term: string, callback: (data: Channel[]) => void) => {
        try {
            debouncedSearchChannels(term, callback);
        } catch (error) {
            // eslint-disable-next-line no-console
            console.error(error);
            callback([]);
        }
    };

    const keyDownHandler = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.stopPropagation();
        }
    };

    return (
        <AsyncSelect
            isMulti={true}
            cacheOptions={true}
            defaultOptions={false}
            loadOptions={channelsLoader}
            onChange={onChange}
            getOptionValue={getOptionValue}
            formatOptionLabel={formatOptionLabel}
            defaultMenuIsOpen={false}
            openMenuOnClick={false}
            isClearable={false}
            placeholder={props.placeholder}
            value={props.channels}
            components={{DropdownIndicator: () => null, IndicatorSeparator: () => null}}
            styles={customStyles}
            menuPortalTarget={document.body}
            menuPosition={'fixed'}
            onKeyDown={keyDownHandler}
        />
    );
}

const customStyles: StylesConfig<any, true> = {
    container: (baseStyles) => ({
        ...baseStyles,
    }),
    control: (baseStyles) => ({
        ...baseStyles,
        minHeight: '46px',
    }),
    menuPortal: (baseStyles) => ({
        ...baseStyles,
        zIndex: 9999,
    }),
    multiValue: (baseStyles) => ({
        ...baseStyles,
        borderRadius: '50px',
    }),
};
