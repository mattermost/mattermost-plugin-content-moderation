// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react';

import {DEFAULT_AGENTS_SYSTEM_PROMPT} from '@/components/admin_settings/agents_constants';

type ModeratorType = 'azure' | 'agents';

const THRESHOLD_OPTIONS = [
    {value: '2', label: 'Low (2)'},
    {value: '4', label: 'Medium (4)'},
    {value: '6', label: 'High (6)'},
] as const;

interface ModeratorConfigValue {
    type: ModeratorType;
    azure_endpoint?: string;
    azure_apiKey?: string;
    azure_threshold?: string;
    agents_system_prompt?: string;
    agents_threshold?: string;
    agents_bot_username?: string;
}

interface ModeratorConfigProps {
    id: string;
    value?: ModeratorConfigValue;
    onChange: (id: string, value: ModeratorConfigValue) => void;
    settings: Record<string, unknown>;
    config?: Record<string, unknown>;
}

// Helper function to create default configuration values
const createDefaultConfig = (type: ModeratorType = 'agents', existingValues?: Partial<ModeratorConfigValue>): ModeratorConfigValue => {
    const defaults: ModeratorConfigValue = {
        type,
        azure_endpoint: '',
        azure_apiKey: '',
        azure_threshold: THRESHOLD_OPTIONS[0].value, // '2'
        agents_system_prompt: DEFAULT_AGENTS_SYSTEM_PROMPT,
        agents_threshold: THRESHOLD_OPTIONS[0].value, // '2'
        agents_bot_username: '',
        ...existingValues,
    };
    return defaults;
};

const ModeratorConfig: React.FC<ModeratorConfigProps> = ({id, value, onChange}) => {
    // Initialize with clean defaults, using any provided values
    const initialConfig = createDefaultConfig(value?.type, value);
    const [currentType, setCurrentType] = useState<ModeratorType>(initialConfig.type);
    const [values, setValues] = useState<ModeratorConfigValue>(initialConfig);

    // Update state when props change
    useEffect(() => {
        if (value && JSON.stringify(value) !== JSON.stringify(values)) {
            const newValues = createDefaultConfig(value.type, value);
            setCurrentType(newValues.type);
            setValues(newValues);
        }
    }, [value]);

    // Notify parent of initial values on component mount
    useEffect(() => {
        if (!value) {
            onChange(id, initialConfig);
        }
    }, [value, onChange, id, initialConfig]);

    // Memoized field change handler to prevent unnecessary re-renders
    const handleFieldChange = useCallback((field: keyof ModeratorConfigValue, fieldValue: string) => {
        let newValues: ModeratorConfigValue;

        if (field === 'type') {
            // When switching type, merge with appropriate defaults for the new type
            newValues = createDefaultConfig(fieldValue as ModeratorType, values);
            setCurrentType(fieldValue as ModeratorType);
        } else {
            newValues = {
                ...values,
                [field]: fieldValue,
            };
        }

        setValues(newValues);
        onChange(id, newValues);
    }, [values, onChange, id]);

    // Memoized render functions to prevent unnecessary re-renders
    const renderAzureSettings = useCallback((settings: ModeratorConfigValue) => {
        const azureEndpoint = settings.azure_endpoint || '';
        const azureApiKey = settings.azure_apiKey || '';
        const azureThreshold = settings.azure_threshold || '2';

        return (
            <>
                <div style={{marginBottom: '16px'}}>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'Azure API Endpoint'}
                    </label>
                    <input
                        type='text'
                        value={azureEndpoint}
                        onChange={(e) => handleFieldChange('azure_endpoint', e.target.value)}
                        placeholder='https://your-resource.cognitiveservices.azure.com/'
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                        }}
                    />
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'The endpoint URL for the Azure AI Content Safety API.'}
                    </p>
                </div>

                <div style={{marginBottom: '16px'}}>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'Azure API Key'}
                    </label>
                    <input
                        type='password'
                        value={azureApiKey}
                        onChange={(e) => handleFieldChange('azure_apiKey', e.target.value)}
                        placeholder='Enter your Azure API key'
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                        }}
                    />
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'Your Azure AI Content Safety API key.'}
                    </p>
                </div>

                <div>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'Moderation Threshold'}
                    </label>
                    <select
                        value={azureThreshold}
                        onChange={(e) => handleFieldChange('azure_threshold', e.target.value)}
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                        }}
                    >
                        {THRESHOLD_OPTIONS.map((option) => (
                            <option
                                key={option.value}
                                value={option.value}
                            >
                                {option.label}
                            </option>
                        ))}
                    </select>
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'Severity threshold for all content categories (Low filters most aggressively).'}
                    </p>
                </div>
            </>
        );
    }, [handleFieldChange]);

    const renderAgentsSettings = useCallback((settings: ModeratorConfigValue) => {
        const agentsSystemPrompt = settings.agents_system_prompt || '';
        const agentsThreshold = settings.agents_threshold || '2';
        const agentsBotUsername = settings.agents_bot_username || '';

        return (
            <>
                <div style={{marginBottom: '16px'}}>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'Agent Bot Username'}
                    </label>
                    <input
                        type='text'
                        value={agentsBotUsername}
                        onChange={(e) => handleFieldChange('agents_bot_username', e.target.value)}
                        placeholder='content-moderation-agent'
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                        }}
                    />
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'Username of the specific agent to use for content moderation. Leave empty to use the default agent.'}
                    </p>
                </div>

                <div style={{marginBottom: '16px'}}>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'System Prompt'}
                    </label>
                    <textarea
                        value={agentsSystemPrompt}
                        onChange={(e) => handleFieldChange('agents_system_prompt', e.target.value)}
                        placeholder='Default prompt will be used when empty'
                        rows={4}
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                            resize: 'vertical',
                        }}
                    />
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'Custom system prompt for the LLM moderation. The default prompt will be used when this field is empty.'}
                    </p>
                </div>

                <div>
                    <label
                        style={{
                            display: 'block',
                            marginBottom: '8px',
                            color: '#3f4350',
                            fontSize: '14px',
                            fontWeight: '600',
                        }}
                    >
                        {'Moderation Threshold'}
                    </label>
                    <select
                        value={agentsThreshold}
                        onChange={(e) => handleFieldChange('agents_threshold', e.target.value)}
                        style={{
                            width: '100%',
                            padding: '8px 12px',
                            border: '1px solid #d1d5db',
                            borderRadius: '4px',
                            fontSize: '14px',
                            boxSizing: 'border-box',
                        }}
                    >
                        {THRESHOLD_OPTIONS.map((option) => (
                            <option
                                key={option.value}
                                value={option.value}
                            >
                                {option.label}
                            </option>
                        ))}
                    </select>
                    <p
                        style={{
                            marginTop: '4px',
                            marginBottom: '0',
                            color: '#6b7280',
                            fontSize: '12px',
                        }}
                    >
                        {'Severity threshold for all content categories (Low filters most aggressively).'}
                    </p>
                </div>
            </>
        );
    }, [handleFieldChange]);

    return (
        <div
            style={{
                border: '1px solid #e6e6e6',
                borderRadius: '4px',
                padding: '24px 16px',
                marginBottom: '24px',
                backgroundColor: '#f9f9f9',
            }}
        >

            <div style={{marginBottom: '24px'}}>
                <label
                    style={{
                        display: 'block',
                        marginBottom: '8px',
                        color: '#3f4350',
                        fontSize: '14px',
                        fontWeight: '600',
                    }}
                >
                    {'Moderation Provider'}
                </label>
                <select
                    value={currentType}
                    onChange={(e) => handleFieldChange('type', e.target.value)}
                    style={{
                        width: '100%',
                        padding: '8px 12px',
                        border: '1px solid #d1d5db',
                        borderRadius: '4px',
                        fontSize: '14px',
                        boxSizing: 'border-box',
                    }}
                >
                    <option value='azure'>{'Azure AI Content Safety'}</option>
                    <option value='agents'>{'Mattermost Agents Plugin'}</option>
                </select>
                <p
                    style={{
                        marginTop: '4px',
                        marginBottom: '0',
                        color: '#6b7280',
                        fontSize: '12px',
                    }}
                >
                    {'Select which content moderation provider to use.'}
                </p>
            </div>

            {currentType === 'azure' && renderAzureSettings(values)}
            {currentType === 'agents' && renderAgentsSettings(values)}
        </div>
    );
};

export default ModeratorConfig;
