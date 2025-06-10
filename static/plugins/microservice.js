import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('microservice', {
    type: 'microservice',
    label: 'Microservice',
    props: [
        { name: 'label', type: 'string', default: 'Service', group: 'label-group' },
        {
            name: 'instanceCount',
            type: 'number',
            default: 3,
            group: 'microservice-group'
        },
        {
            name: 'instanceSize',
            type: 'string',
            default: 'medium',
            group: 'microservice-group'
        },
        {
            name: 'scalingStrategy',
            type: 'string',
            default: 'auto',
            group: 'microservice-group'
        },
        {
            name: 'apiVersion',
            type: 'string',
            default: 'v1',
            group: 'microservice-group'
        }
    ]
});

