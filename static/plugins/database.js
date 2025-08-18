import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('database', {
    type: 'database',
    label: 'Database',
    props: [
        { name: 'label', type: 'string', default: 'Database', group: 'label-group' },
        { name: 'replication', type: 'number', default: 1, group: 'db-group' },
        { name: 'maxRPS', type: 'number', default: 1000, group: 'db-group' },
        { name: 'baseLatencyMs', type: 'number', default: 10, group: 'db-group' }
    ]
});
