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
        this.startChatBtn = document.getElementById('start-chat');
        this.chatElement = document.getElementById('chat-box');
        this.chatTextField = document.getElementById('chat-message-box');
        this.chatMessages = document.getElementById('messages');
        this.chatLoadingIndicator = document.getElementById('loading-indicator');
        this.level = window.levelData;
        this.ws = null;
        this.plugins = PluginRegistry.getAll()
        this.createDesignBtn = document.getElementById('create-design-button');
        this.learnMoreBtn = document.getElementById('learn-more-button');
        this.tabs = document.getElementsByClassName('tabinput');

        console.log(this.tabs)
        this._reconnectDelay = 1000;
        this._maxReconnectDelay = 15000;
        this._reconnectTimer = null;

        this.initEventHandlers();
    }

    initEventHandlers() {
        const requirementstab = this.tabs[1];
        const designtab = this.tabs[1];
        const resourcestab = this.tabs[2];

        this.learnMoreBtn.addEventListener('click', () => {
            requirementstab.checked = false;
            resourcestab.checked = true;
        });

        this.createDesignBtn.addEventListener('click', () => {
            requirementstab.checked = false;
            designtab.checked = true;
        });

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
        this.startChatBtn.addEventListener('click', () => {
            const scheme = location.protocol === "https:" ? "wss://" : "ws://";

            this.ws = new WebSocket(scheme + location.host + "/ws");

            this.ws.onopen = () => {
                this.ws.send(JSON.stringify({
                    'designPayload': JSON.stringify(this.exportDesign()),
                    'message': ''
                }));
            }

            this.ws.onmessage = (e) => {
                this.chatLoadingIndicator.style.display = 'none';
                this.chatTextField.disabled = false;
                this.chatTextField.focus();
                const message = document.createElement('p');
                message.innerHTML = e.data;
                message.className = "other";
                this.chatMessages.insertBefore(message, this.chatLoadingIndicator)
            }

            this.ws.onerror = (err) => {
                console.log("ws error:", err);
                this._scheduleReconnect()
            };

            this.ws.onclose = () => {
                console.log("leaving chat...")
                this.ws = null;
                this._sentJoin = false;
                delete this.players[this.pageData.username]
                this._scheduleReconnect()
            }


        })
        this.chatTextField.addEventListener('keydown', (e) => {

            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                console.log('you sent a message')

                const message = document.createElement('p');
                message.innerHTML = this.chatTextField.value;
                message.className = "me";
                this.chatMessages.insertBefore(message, this.chatLoadingIndicator);


                this.ws.send(JSON.stringify({
                    'message': this.chatTextField.value,
                    'designPayload': JSON.stringify(this.exportDesign()),
                }));

                this.chatTextField.value = '';
                this.chatLoadingIndicator.style.display = 'block';
            }

        })
        // start a ws connection
        // onopen, send the payload

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

        this.runButton.addEventListener('click', async () => {
            const designData = this.exportDesign();

            // Try to get level info from URL or page context
            const levelInfo = this.getLevelInfo();

            const requestBody = {
                design: designData,
                ...levelInfo
            };

            console.log('Sending design to simulation:', JSON.stringify(requestBody));

            // Disable button and show loading state
            this.runButton.disabled = true;
            this.runButton.textContent = 'Running Simulation...';

            try {
                const response = await fetch('/simulate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestBody)
                });

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const result = await response.json();

                if (result.Success) {
                    console.log('Simulation successful:', result);
                    this.showResults(result);
                } else {
                    console.error('Simulation failed:', result.Error);
                    this.showError(result.Error || 'Simulation failed');
                }

            } catch (error) {
                console.error('Network error:', error);
                this.showError('Failed to run simulation: ' + error.message);
            } finally {
                // Re-enable button
                this.runButton.disabled = false;
                this.runButton.textContent = 'Test Design';
            }
        });

        this.canvas.addEventListener('click', (e) => {
            // If this is part of a double-click sequence (detail > 1), ignore it
            if (e.detail > 1) {
                return;
            }
            
            if (this.connectionStart) {
                this.connectionStart.group.classList.remove('selected');
                this.connectionStart = null;
            }
            
            // Don't hide props panel if clicking on it
            if (!this.nodePropsPanel.contains(e.target)) {
                this.hidePropsPanel();
            }
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

        // Prevent props panel from closing when clicking inside it
        this.nodePropsPanel.addEventListener('click', (e) => {
            e.stopPropagation();
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

        // Get the node's actual screen position using getBoundingClientRect
        const nodeRect = node.group.getBoundingClientRect();
        const containerRect = this.canvasContainer.getBoundingClientRect();
        const panelWidth = 220; // From CSS: #node-props-panel width
        const panelHeight = 400; // Estimated height for boundary checking
        
        // Try to position dialog to the right of the node
        let dialogX = nodeRect.right + 10;
        let dialogY = nodeRect.top;
        
        // Check if dialog would go off the right edge of the screen
        if (dialogX + panelWidth > window.innerWidth) {
            // Position to the left of the node instead
            dialogX = nodeRect.left - panelWidth - 10;
        }
        
        // Check if dialog would go off the bottom of the screen
        if (dialogY + panelHeight > window.innerHeight) {
            // Move up to keep it visible
            dialogY = window.innerHeight - panelHeight - 10;
        }
        
        // Ensure dialog doesn't go above the top of the screen
        if (dialogY < 10) {
            dialogY = 10;
        }
        
        panel.style.left = dialogX + 'px';
        panel.style.top = dialogY + 'px';

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
        
        // Trigger smooth animation
        setTimeout(() => {
            panel.classList.add('visible');
        }, 10);
    }

    hidePropsPanel() {
        const panel = this.nodePropsPanel;
        panel.classList.remove('visible');
        
        // Hide after animation completes
        setTimeout(() => {
            panel.style.display = 'none';
        }, 200);
        
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

        return {
            nodes,
            connections,
            level: JSON.parse(this.level),
            availableComponents: JSON.stringify(this.plugins)
        };
    }

    getLevelInfo() {
        // Try to extract level info from URL path like /play/url-shortener
        const pathParts = window.location.pathname.split('/');
        if (pathParts.length >= 3 && pathParts[1] === 'play') {
            const levelId = decodeURIComponent(pathParts[2]);
            return {
                levelId: levelId
            };
        }
        return {};
    }

    showResults(result) {
        const metrics = result.Metrics;
        let message = '';

        // Level validation results
        if (result.LevelName) {
            if (result.Passed) {
                message += `Level "${result.LevelName}" PASSED!\n`;
                message += `Score: ${result.Score}/100\n\n`;
            } else {
                message += `Level "${result.LevelName}" FAILED\n`;
                message += `Score: ${result.Score}/100\n\n`;
            }

            // Add detailed feedback
            if (result.Feedback && result.Feedback.length > 0) {
                message += result.Feedback.join('\n') + '\n\n';
            }
        } else {
            message += `Simulation Complete!\n\n`;
        }

        // Performance metrics
        message += `Performance Metrics:\n`;
        message += `• Throughput: ${metrics.throughput} req/sec\n`;
        message += `• Avg Latency: ${metrics.latency_avg}ms\n`;
        message += `• Availability: ${metrics.availability.toFixed(1)}%\n`;
        message += `• Monthly Cost: $${metrics.cost_monthly}\n\n`;
        message += `Timeline: ${result.Timeline.length} ticks simulated`;

        alert(message);

        // TODO: Later replace with redirect to results page or modal
        console.log('Full simulation data:', result);
    }

    showError(errorMessage) {
        alert(`Simulation Error:\n\n${errorMessage}\n\nPlease check your design and try again.`);
    }

    _scheduleReconnect() {
        if (this._stopped) return;

        if (this._reconnectTimer) {
            clearTimeout(this._reconnectTimer)
            this._reconnectTimer = null;
        }

        const jitter = this._reconnectDelay * (Math.random() * 0.4 - 0.2);
        const delay = Math.max(250, Math.min(this._maxReconnectDelay, this._reconnectDelay + jitter));
        console.log(`Reconnecting websocket...`)

        this._reconnectTimer = setTimeout(() => {
            this._reconnectTimer = null;
            this._initWebSocket();
        }, delay);

        this._reconnectDelay = Math.min(this._maxReconnectDelay, Math.round(this._reconnectDelay * 1.8));
    }

}
