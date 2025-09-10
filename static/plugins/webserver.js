import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('webserver', {
    type: 'webserver',
    label: 'Web Server',
    props: [
        { name: 'label', type: 'string', default: 'Web Server', group: 'label-group' },
        { name: 'rpsCapacity', type: 'number', default: 200, group: 'compute-group' },
        { name: 'baseLatencyMs', type: 'number', default: 20, group: 'compute-group' }
    ]
});
