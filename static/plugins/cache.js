import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('cache', {
    type: 'cache',
    label: 'Cache',
    props: [
        { name: 'label', type: 'string', default: 'Cache', group: 'label-group' },
        { name: 'cacheTTL', type: 'number', default: 60, group: 'cache-group' },
        { name: 'maxEntries', type: 'number', default: 100000, group: 'cache-group' },
        { name: 'evictionPolicy', type: 'string', default: 'LRU', group: 'cache-group' }
    ]
});

