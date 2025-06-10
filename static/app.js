import { ComponentNode } from './node.js'
import { generateNodeId, createSVGElement, generateDefaultProps } from './utils.js';
import './plugins/user.js';
import './plugins/webserver.js';
import './plugins/cache.js';
import './plugins/loadbalancer.js';
import './plugins/database.js';
import './plugins/messageQueue.js';
import './plugins/cdn.js';
import './plugins/microservice.js';
import './plugins/datapipeline.js';
import './plugins/monitorAlerting.js';
import './plugins/thirdPartyService.js';
import { PluginRegistry } from './pluginRegistry.js';

export class CanvasApp {
    constructor() {
        this.placedComponents = [];
        this.connections = [];
        this.componentSize = { width: 120, height: 40 };
        this.arrowMode = false;
        this.connectionStart = null;
        this.pendingConnection = null;
        this.activeNode = null;
        this.selectedConnection = null;

        this.sidebar = document.getElementById('sidebar');
        this.arrowToolBtn = document.getElementById('arrow-tool-btn');
        this.canvasContainer = document.getElementById('canvas-container');
        this.canvas = document.getElementById('canvas');
        this.runButton = document.getElementById('run-button');
        this.nodePropsPanel = document.getElementById('node-props-panel');
        this.propsSaveBtn = document.getElementById('node-props-save');
        this.labelGroup = document.getElementById('label-group');
        this.dbGroup = document.getElementById('db-group');
        this.cacheGroup = document.getElementById('cache-group');
        this.selectedNode = null;
        this.computeGroup = document.getElementById('compute-group');
        this.lbGroup = document.getElementById('lb-group');
        this.mqGroup = document.getElementById('mq-group');


        this.initEventHandlers();
    }

    initEventHandlers() {
        this.arrowToolBtn.addEventListener('click', () => {
            this.arrowMode = !this.arrowMode;
            if (this.arrowMode) {
                this.arrowToolBtn.classList.add('active');
                this.hidePropsPanel();
            } else {
                this.arrowToolBtn.classList.remove('active');
                if (this.connectionStart) {
                    this.connectionStart.group.classList.remove('selected');
                    this.connectionStart = null;
                }
            }
        });
        this.sidebar.addEventListener('dragstart', (e) => {
            const type = e.target.getAttribute('data-type');
            const plugin = PluginRegistry.get(type);

            if (!plugin) return;

            e.dataTransfer.setData('text/plain', type)
        });
        this.sidebar.addEventListener('dragend', (e) => {
            if (e.target.classList.contains('component-icon')) {
                e.target.classList.remove('dragging');
            }
        });

        this.canvasContainer.addEventListener('dragover', (e) => e.preventDefault());
        this.canvasContainer.addEventListener('drop', (e) => {
            const type = e.dataTransfer.getData('text/plain');
            const plugin = PluginRegistry.get(type);
            if (!plugin) return;

            const pt = this.canvas.createSVGPoint();
            pt.x = e.clientX;
            pt.y = e.clientY;

            const svgP = pt.matrixTransform(this.canvas.getScreenCTM().inverse());
            const x = svgP.x - this.componentSize.width / 2;
            const y = svgP.y - this.componentSize.height / 2;

            const props = generateDefaultProps(plugin);
            const node = new ComponentNode(type, x, y, this, props);
            node.x = x;
            node.y = y;
        });

        this.runButton.addEventListener('click', () => {
            const designData = this.exportDesign();
            console.log(JSON.stringify(designData))
        });

        this.canvas.addEventListener('click', () => {
            if (this.connectionStart) {
                this.connectionStart.group.classList.remove('selected');
                this.connectionStart = null;
            }
            this.hidePropsPanel();
            this.clearSelection();
        });

        this.propsSaveBtn.addEventListener('click', () => {
            if (!this.activeNode) return;

            const node = this.activeNode;
            const panel = this.nodePropsPanel;
            const plugin = PluginRegistry.get(node.type);

            if (!plugin || !plugin.props) {
                this.hidePropsPanel();
                return;
            }

            // Loop through plugin-defined props and update the node
            for (const prop of plugin.props) {
                const input = panel.querySelector(`[name='${prop.name}']`);
                if (!input) continue;

                let value;
                if (prop.type === 'number') {
                    value = parseFloat(input.value);
                    if (isNaN(value)) value = prop.default ?? 0;
                } else {
                    value = input.value;
                }

                node.props[prop.name] = value;
                if (prop.name === 'label') {
                    node.updateLabel(value);
                }
            }

            this.hidePropsPanel();
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Backspace' || e.key === 'Delete') {
                if (this.selectedConnection) {
                    this.canvas.removeChild(this.selectedConnection.line);
                    this.canvas.removeChild(this.selectedConnection.text);
                    const index = this.connections.indexOf(this.selectedConnection);
                    if (index !== -1) this.connections.splice(index, 1);
                    this.selectedConnection = null;
                } else if (this.selectedNode) {
                    this.canvas.removeChild(this.selectedNode.group);
                    this.placedComponents = this.placedComponents.filter(n => n !== this.selectedNode);
                    this.connections = this.connections.filter(conn => {
                        if (conn.start === this.selectedNode || conn.end === this.selectedNode) {
                            this.canvas.removeChild(conn.line);
                            this.canvas.removeChild(conn.text);
                            return false;
                        }
                        return true;
                    });
                    this.selectedNode = null;
                    this.activeNode = null;
                    this.hidePropsPanel();
                }
            }
        });
    }

    showPropsPanel(node) {
        this.activeNode = node;
        const plugin = PluginRegistry.get(node.type);
        const panel = this.nodePropsPanel;

        if (!plugin || this.arrowMode) {
            this.hidePropsPanel();
            return;
        }

        const bbox = node.group.getBBox();
        const ctm = node.group.getCTM();
        const screenX = ctm.e + bbox.x;
        const screenY = ctm.f + bbox.y + bbox.height;
        panel.style.left = (screenX + this.canvasContainer.getBoundingClientRect().left) + 'px';
        panel.style.top = (screenY + this.canvasContainer.getBoundingClientRect().top) + 'px';

        // Hide all groups first
        const allGroups = panel.querySelectorAll('.prop-group, #label-group, #compute-group, #lb-group');
        allGroups.forEach(g => g.style.display = 'none');

        const shownGroups = new Set();

        for (const prop of plugin.props) {
            const group = panel.querySelector(`[data-group='${prop.group}']`);
            const input = panel.querySelector(`[name='${prop.name}']`);

            // Show group once
            if (group && !shownGroups.has(group)) {
                group.style.display = 'block';
                shownGroups.add(group);
            }

            // Set value
            if (input) {
                input.value = node.props[prop.name] ?? prop.default;
            }
        }

        this.propsSaveBtn.disabled = false;
        panel.style.display = 'block';
    }

    hidePropsPanel() {
        this.nodePropsPanel.style.display = 'none';
        this.propsSaveBtn.disabled = true;
        this.activeNode = null;
    }

    updateConnectionsFor(movedNode) {
        this.connections.forEach(conn => {
            if (conn.start === movedNode || conn.end === movedNode) {
                conn.updatePosition();
            }
        });
    }

    clearSelection() {
        if (this.selectedConnection) {
            this.selectedConnection.deselect();
            this.selectedConnection = null;
        }

        if (this.selectedNode) {
            this.selectedNode.deselect();
            this.selectedNode = null;
            this.hidePropsPanel();
        }
    }

    exportDesign() {
        const nodes = this.placedComponents
            .filter(n => n.type !== 'user')
            .map(n => {
                const plugin = PluginRegistry.get(n.type);
                const result = {
                    id: n.id,
                    type: n.type,
                    position: { x: n.x, y: n.y },
                    props: {}
                };

                plugin?.props?.forEach(p => {
                    result.props[p.name] = n.props[p.name];
                });

                return result;
            });

        const connections = this.connections.map(c => ({
            source: c.start.id,
            target: c.end.id,
            label: c.label || '',
            direction: c.direction,
            protocol: c.protocol || '',
            tls: !!c.tls,
            capacity: c.capacity || 1000
        }));

        return { nodes, connections };
    }
}
