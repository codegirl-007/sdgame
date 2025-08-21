export class PluginRegistry {
    static plugins = {}

    static register(type, plugin) {
        this.plugins[type] = plugin;
    }

    static get(type) {
        return this.plugins[type];
    }

    static getAll() {
        return Object.keys(this.plugins)
    }
}
