import { generateNodeId, createSVGElement } from './utils.js';

export class Connection {
    constructor(startNode, endNode, label, protocol, app) {
        this.start = startNode;
        this.end = endNode;
        this.app = app;
        this.label = label;
        this.protocol = protocol;
        this.direction = "forward";
        this.line = createSVGElement('line', {
            stroke: '#ccc', 'stroke-width': 2, 'marker-end': 'url(#arrowhead)'
        });
        this.hitbox = createSVGElement('circle', {
            r: 12,
            fill: 'transparent',
            cursor: 'pointer',
        });
        this.app.canvas.appendChild(this.hitbox);
        this.text = createSVGElement('text', {
            'text-anchor': 'middle', 'font-size': 12, fill: '#ccc'
        });
        this.text.textContent = label;
        this.protocolText = createSVGElement('text', {
            'text-ancor': 'middle',
            'font-size': 10,
            fill: '#888'
        });
        this.protocolText.textContent = this.protocol || '';
        app.canvas.appendChild(this.line);
        app.canvas.appendChild(this.text);
        app.canvas.appendChild(this.protocolText)
        this.updatePosition();

        this.selected = false;
        this.line.addEventListener('click', (e) => {
            e.stopPropagation();
            this.app.clearSelection();
            this.select();
        });

        this.line.addEventListener('dblclick', (e) => {
            e.stopPropagation();
            this.toggleDirection();
        })
    }

    updatePosition() {
        const s = this.start.getConnectionPointToward(this.end);
        const e = this.end.getConnectionPointToward(this.start);

        this.line.setAttribute('x1', s.x);
        this.line.setAttribute('y1', s.y);
        this.line.setAttribute('x2', e.x);
        this.line.setAttribute('y2', e.y);

        // update text position (midpoint of the line)
        const midX = (s.x + e.x) / 2;
        const midY = (s.y + e.y) / 2;
        this.text.setAttribute('x', midX);
        this.text.setAttribute('y', midY - 6);

        // update arrowheads based on direction
        if (this.direction === 'forward') {
            this.line.setAttribute('marker-start', '');
            this.line.setAttribute('marker-end', 'url(#arrowhead-end)');
        } else if (this.direction === 'backward') {
            this.line.setAttribute('marker-start', 'url(#arrowhead-start)');
            this.line.setAttribute('marker-end', '');
        } else if (this.direction === 'bidirectional') {
            this.line.setAttribute('marker-start', 'url(#arrowhead-start)');
            this.line.setAttribute('marker-end', 'url(#arrowhead-end)');
        }

        // Hitbox position (depends on direction)
        let hbX, hbY;
        if (this.direction === 'forward') {
            hbX = e.x;
            hbY = e.y;
        } else if (this.direction === 'backward') {
            hbX = s.x;
            hbY = s.y;
        } else {
            hbX = midX;
            hbY = midY;
        }

        this.hitbox.setAttribute('cx', hbX);
        this.hitbox.setAttribute('cy', hbY);

        this.protocolText.setAttribute('x', midX);
        this.protocolText.setAttribute('y', midY + 12);
    }

    toggleDirection() {
        const order = ['forward', 'backward', 'bidirectional'];
        const currentIndex = order.indexOf(this.direction);
        const nextIndex = (currentIndex + 1) % order.length;
        this.direction = order[nextIndex]
        this.updatePosition();
    }

    select() {
        this.app.clearSelection();
        this.selected = true;
        this.line.setAttribute('stroke', '#007bff');
        this.line.setAttribute('stroke-width', 3);
        this.app.selectedConnection = this;
    }

    deselect() {
        this.selected = false;
        this.line.setAttribute('stroke', '#333');
        this.line.setAttribute('stroke-width', 2);
    }
}
