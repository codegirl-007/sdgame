import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('loadBalancer', {
    type: 'loadBalancer',
    label: 'Load Balancer',
    props: [
        { name: 'label', type: 'string', default: 'Load Balancer', group: 'label-group' },
        { name: 'algorithm', type: 'string', default: 'round-robin', group: 'lb-group' }
    ]
});
