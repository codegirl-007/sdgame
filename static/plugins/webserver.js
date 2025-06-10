import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('webserver', {
    type: 'webserver',
    label: 'Web Server',
    props: [
        { name: 'label', type: 'string', default: 'Web Server', group: 'label-group' },
        { name: 'instanceSize', type: 'string', default: 'medium', group: 'compute-group' }
    ]
});
