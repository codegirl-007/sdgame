import { PluginRegistry } from "../pluginRegistry.js";

PluginRegistry.register('monitoring/alerting', {
    type: 'monitoring/alerting',
    label: 'Monitoring/Alerting',
    props: [
        { name: 'label', type: 'string', default: 'monitor', group: 'label-group' },
        { name: 'tool', type: 'string', default: 'Prometheus', group: 'monitor-group' },
        { name: 'alertMetric', type: 'string', default: 'latency', group: 'monitor-group' },
        { name: 'thresholdValue', type: 'number', default: 80, group: 'monitor-group' },
        { name: 'thresholdUnit', type: 'string', default: 'ms', group: 'monitor-group' }
    ]
});
