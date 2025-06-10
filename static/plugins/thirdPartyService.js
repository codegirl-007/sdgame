import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('third party service', {
    type: 'third party service',
    label: 'Third-Party Service',
    props: [
        { name: 'label', type: 'string', default: 'third party service', group: 'label-group' },
        { name: 'provider', type: 'string', default: 'Stripe', group: 'third-party-group' },
        { name: 'latency', type: 'number', default: 200, group: 'third-party-group' }
    ]
});
