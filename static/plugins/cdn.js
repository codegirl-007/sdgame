import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('cdn', {
    type: 'cdn',
    label: 'CDN',
    props: [
        { name: 'label', type: 'string', default: 'CDN', group: 'label-group' },
        { name: 'ttl', type: 'number', default: 3600, group: 'cdn-group' },
        {
            name: 'geoReplication',
            type: 'string',
            default: 'global',
            group: 'cdn-group'
        },
        {
            name: 'cachingStrategy',
            type: 'string',
            default: 'cache-first',
            group: 'cdn-group'
        },
        {
            name: 'compression',
            type: 'string',
            default: 'brotli',
            group: 'cdn-group'
        },
        {
            name: 'http2',
            type: 'string',
            default: 'enabled',
            group: 'cdn-group'
        }
    ]
});
