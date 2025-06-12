import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('microservice', {
    type: 'microservice',
    label: 'Microservice',
    props: [
        { name: 'label', type: 'string', default: 'Service', group: 'label-group' },
        { name: 'instanceCount', type: 'number', default: 3, group: 'microservice-group' },
        { name: 'cpu', type: 'number', default: 2, group: 'microservice-group' },
        { name: 'ramGb', type: 'number', default: 4, group: 'microservice-group' },
        { name: 'rpsCapacity', type: 'number', default: 150, group: 'microservice-group' },
        { name: 'monthlyUsd', type: 'number', default: 18, group: 'microservice-group' },
        { name: 'scalingStrategy', type: 'string', default: 'auto', group: 'microservice-group' },
        { name: 'apiVersion', type: 'string', default: 'v1', group: 'microservice-group' }
    ]
});
