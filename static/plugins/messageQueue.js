import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('messageQueue', {
    type: 'messageQueue',
    label: 'Message Queue',
    props: [
        { name: 'label', type: 'string', default: 'MQ', group: 'label-group' },
        { name: 'queueCapacity', type: 'number', default: 10000, group: 'mq-group' },
        { name: 'retentionSeconds', type: 'number', default: 600, group: 'mq-group' },
        { name: 'processingRate', type: 'number', default: 100, group: 'mq-group' }
    ]
});
