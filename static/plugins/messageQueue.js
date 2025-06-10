import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('messageQueue', {
    type: 'messageQueue',
    label: 'Message Queue',
    props: [
        { name: 'label', type: 'string', default: 'MQ', group: 'label-group' },
        { name: 'maxSize', type: 'number', default: 10000, group: 'mq-group' },
        { name: 'retentionSeconds', type: 'number', default: 600, group: 'mq-group' }
    ]
});
