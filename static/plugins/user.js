import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('user', {
    type: 'user',
    label: 'User',
    isVisualOnly: true,
    props: [
        { name: 'label', type: 'string', default: 'User', group: 'label-group' }
    ]
});
