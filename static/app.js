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
import { initializeObservers } from './observers.js';
import { 
    CommandInvoker,
    SwitchToResourcesTabCommand,
    SwitchToDesignTabCommand,
    ToggleArrowModeCommand,
    StartChatCommand,
    SendChatMessageCommand,
    HandleDragStartCommand,
    HandleDragEndCommand,
    DropComponentCommand,
    RunSimulationCommand,
    HandleCanvasClickCommand,
    SaveNodePropertiesCommand,
    DeleteSelectionCommand
} from './commands.js';

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

        // Initialize observer system (alongside existing event handling)
        const observers = initializeObservers(this.nodePropsPanel, this.propsSaveBtn);
        this.propertiesPanelSubject = observers.propertiesPanel;
        this.nodeSelectionSubject = observers.nodeSelection;
        this.connectionModeSubject = observers.connectionMode;

        // Initialize command system
        this.commandInvoker = new CommandInvoker(this);

        this.initEventHandlers();
    }

    initEventHandlers() {
        const requirementstab = this.tabs[1];
        const designtab = this.tabs[1];
        const resourcestab = this.tabs[2];

        this.learnMoreBtn.addEventListener('click', () => {
            this.commandInvoker.execute(new SwitchToResourcesTabCommand());
        });

        this.createDesignBtn.addEventListener('click', () => {
            this.commandInvoker.execute(new SwitchToDesignTabCommand());
        });

        this.arrowToolBtn.addEventListener('click', () => {
            this.commandInvoker.execute(new ToggleArrowModeCommand());
        });
        this.startChatBtn.addEventListener('click', () => {
            this.commandInvoker.execute(new StartChatCommand());
        });
        this.chatTextField.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                console.log('you sent a message');
                
                const message = this.chatTextField.value;
                if (message.trim()) {
                    this.commandInvoker.execute(new SendChatMessageCommand(message));
                }
            }
        });

        this.sidebar.addEventListener('dragstart', (e) => {
            this.commandInvoker.execute(new HandleDragStartCommand(e));
        });
        this.sidebar.addEventListener('dragend', (e) => {
            this.commandInvoker.execute(new HandleDragEndCommand(e));
        });

        this.canvasContainer.addEventListener('dragover', (e) => e.preventDefault());
        this.canvasContainer.addEventListener('drop', (e) => {
            this.commandInvoker.execute(new DropComponentCommand(e));
        });

        this.runButton.addEventListener('click', () => {
            this.commandInvoker.execute(new RunSimulationCommand());
        });

        this.canvas.addEventListener('click', (e) => {
            this.commandInvoker.execute(new HandleCanvasClickCommand(e));
        });

        this.propsSaveBtn.addEventListener('click', () => {
            this.commandInvoker.execute(new SaveNodePropertiesCommand());
        });

        // Prevent props panel from closing when clicking inside it
        this.nodePropsPanel.addEventListener('click', (e) => {
            e.stopPropagation();
        });

        document.addEventListener('keydown', (e) => {
            this.commandInvoker.execute(new DeleteSelectionCommand(e.key));
        });
    }





    updateConnectionsFor(movedNode) {
        this.connections.forEach(conn => {
            if (conn.start === movedNode || conn.end === movedNode) {
                conn.updatePosition();
            }
        });
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
