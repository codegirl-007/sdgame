import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('monitoring/alerting', {
    type: 'monitoring/alerting',
    label: 'Monitoring/Alerting',
    props: [
        { name: 'label', type: 'string', default: 'monitor', group: 'label-group' },
        { name: 'tool', type: 'string', default: 'Prometheus', group: 'monitor-group' },
        { name: 'alertThreshold', type: 'number', default: 80, group: 'monitor-group' }
    ]
});
