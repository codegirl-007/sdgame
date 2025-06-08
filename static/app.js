import { generateNodeId, createSVGElement } from './utils.js';
import { ComponentNode } from './node.js'
import { Connection } from './connection.js'

export class CanvasApp {
    constructor() {
        this.placedComponents = [];
        this.connections = [];
        this.componentSize = { width: 120, height: 40 };
        this.arrowMode = false;
        this.connectionStart = null;
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

        this.placeholderText = createSVGElement('text', {
            x: '50%',
            y: '50%',
            'text-anchor': 'middle',
            'dominant-baseline': 'middle',
            fill: '#444',
            'font-size': 18,
            'pointer-events': 'none'
        });
        this.placeholderText.textContent = 'Drag and drop elements to start building your system. Press backspace or delete to remove elements.';
        this.canvas.appendChild(this.placeholderText);
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
            if (e.target.classList.contains('component-icon')) {
                e.dataTransfer.setData('text/plain', e.target.getAttribute('data-type'));
                e.target.classList.add('dragging');
            }
        });
        this.sidebar.addEventListener('dragend', (e) => {
            if (e.target.classList.contains('component-icon')) {
                e.target.classList.remove('dragging');
            }
        });

        this.canvasContainer.addEventListener('dragover', (e) => e.preventDefault());
        this.canvasContainer.addEventListener('drop', (e) => {
            e.preventDefault();
            const type = e.dataTransfer.getData('text/plain');
            const pt = this.canvas.createSVGPoint();
            pt.x = e.clientX;
            pt.y = e.clientY;
            const svgP = pt.matrixTransform(this.canvas.getScreenCTM().inverse());
            const x = svgP.x - this.componentSize.width / 2;
            const y = svgP.y - this.componentSize.height / 2;
            const node = new ComponentNode(type, x, y, this);
            node.x = x;
            node.y = y;
            this.placedComponents.push(node)
            if (this.placeholderText) {
                this.placeholderText.remove();
                this.placeholderText = null;
            }
        });

        this.runButton.addEventListener('click', () => {
            const designData = this.exportDesign();
            alert(JSON.stringify(designData));
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
            const nodeObj = this.activeNode;
            const panel = this.nodePropsPanel;
            const newLabel = panel.querySelector("input[name='label']").value;
            nodeObj.updateLabel(newLabel);
            if (nodeObj.type === 'Database') {
                nodeObj.props.replication = parseInt(panel.querySelector("input[name='replication']").value, 10);
            }
            if (nodeObj.type === 'CacheStandard' || nodeObj.type === 'CacheLarge') {
                nodeObj.props.cacheTTL = parseInt(panel.querySelector("input[name='cacheTTL']").value, 10);
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

    handleConnectionClick(nodeObj) {
        if (!this.connectionStart) {
            this.connectionStart = nodeObj;
            nodeObj.group.classList.add('selected');
        } else if (this.connectionStart === nodeObj) {
            this.connectionStart.group.classList.remove('selected');
            this.connectionStart = null;
        } else {
            const defaultLabel = 'Read traffic';
            const label = prompt('Enter connection label:', defaultLabel);
            const protocol = prompt('Protocol (e.g. HTTP, gRPC, Kafka:', 'HTTP')
            if (label && protocol) {
                const conn = new Connection(this.connectionStart, nodeObj, label, protocol, this);
                conn.protocol = protocol;
                conn.protocolText.textContent = protocol;
                this.connections.push(conn);
            }
            this.connectionStart.group.classList.remove('selected');
            this.connectionStart = null;
        }
    }

    showPropsPanel(nodeObj) {
        this.activeNode = nodeObj;
        const panel = this.nodePropsPanel;

        // Position the panel (optional, or you can use fixed top/right)
        const bbox = nodeObj.group.getBBox();
        const ctm = nodeObj.group.getCTM();
        const screenX = ctm.e + bbox.x;
        const screenY = ctm.f + bbox.y + bbox.height;
        panel.style.left = (screenX + this.canvasContainer.getBoundingClientRect().left) + 'px';
        panel.style.top = (screenY + this.canvasContainer.getBoundingClientRect().top) + 'px';

        // Always show label group
        this.labelGroup.style.display = 'block';
        panel.querySelector("input[name='label']").value = nodeObj.props.label;

        // Show DB fields if it's a Database
        this.dbGroup.style.display = nodeObj.type === 'Database' ? 'block' : 'none';
        if (nodeObj.type === 'Database') {
            this.dbGroup.querySelector("input[name='replication']").value = nodeObj.props.replication;
        }

        // Show cache fields if it's a cache
        const isCache = nodeObj.type === 'CacheStandard' || nodeObj.type === 'CacheLarge';
        this.cacheGroup.style.display = isCache ? 'block' : 'none';
        if (isCache) {
            this.cacheGroup.querySelector("input[name='cacheTTL']").value = nodeObj.props.cacheTTL;
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
        const nodes = this.placedComponents.map(n => ({
            id: n.id,
            type: n.type,
            props: n.props,
            position: {
                x: n.x,
                y: n.y
            }
        }));

        const connections = this.connections.map(c => ({
            source: c.start.id,
            target: c.end.id,
            label: c.label || '',
            direction: c.direction,
            protocol: c.protocol || ''
        }));

        return { nodes, connections };
    }
}
