import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('data pipeline', {
    type: 'data pipeline',
    label: 'Data Pipeline',
    props: [
        { name: 'label', type: 'string', default: 'pipeline', group: 'label-group' },
        { name: 'batchSize', type: 'number', default: 500, group: 'pipeline-group' },
        { name: 'transformation', type: 'string', default: 'map', group: 'pipeline-group' }
    ]
});
