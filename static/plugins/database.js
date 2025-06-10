import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('database', {
    type: 'database',
    label: 'Database',
    props: [
        { name: 'label', type: 'string', default: 'Database', group: 'label-group' },
        { name: 'replication', type: 'number', default: 1, group: 'db-group' }
    ]
});
