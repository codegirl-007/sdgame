import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('webserver', {
    type: 'webserver',
    label: 'Web Server',
    props: [
        { name: 'label', type: 'string', default: 'Web Server', group: 'label-group' },
        { name: 'cpu', type: 'number', default: 2, group: 'compute-group' },
        { name: 'ramGb', type: 'number', default: 4, group: 'compute-group' },
        { name: 'rpsCapacity', type: 'number', default: 200, group: 'compute-group' },
        { name: 'monthlyCostUsd', type: 'number', default: 20, group: 'compute-group' }
    ]
});
