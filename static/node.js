import { Connection } from './connection.js';
import { generateNodeId, createSVGElement } from './utils.js';

export class ComponentNode {
	constructor(type, x, y, app, props = {}) {
		this.id = generateNodeId();
		this.type = type;
		this.app = app;
		this.props = {
			label: type,
			replication: 1,
			cacheTTL: 0,
			instanceSize: 'medium',
			...props
		};

		this.group = createSVGElement('g', { class: 'dropped', 'data-type': type });

		const rect = createSVGElement('rect', {
			x,
			y,
			width: 0, // will be updated after measuring text
			height: app.componentSize.height,
			fill: '#121212',
			stroke: '#00ff88',
			'stroke-width': 1,
			rx: 4,
			ry: 4
		});

		this.text = createSVGElement('text', {
			x: x + app.componentSize.width / 2,
			y: y + app.componentSize.height / 2 + 5,
			'text-anchor': 'middle',
			'font-size': 16,
			fill: '#ccc'
		});

		this.text.textContent = this.props.label;

		// Temporarily add text to canvas to measure its width
		app.canvas.appendChild(this.text);
		const textWidth = this.text.getBBox().width;
		const padding = 20;
		const finalWidth = textWidth + padding;

		// Update rect width and center text
		rect.setAttribute('width', finalWidth);
		this.text.setAttribute('x', x + finalWidth / 2);

		// Clean up temporary text
		app.canvas.removeChild(this.text);

		this.group.appendChild(rect);
		this.group.appendChild(this.text);
		this.group.__nodeObj = this;

		this.initDrag();

		this.group.addEventListener('click', (e) => {
			e.stopPropagation();
			if (app.arrowMode) {
				Connection.handleClick(this, app);
			} else {
				// Use observer to notify node selection
				app.nodeSelectionSubject.notifyNodeSelected(this);
				app.selectedNode = this; // Keep app state in sync for now
			}
		});

		this.group.addEventListener('dblclick', (e) => {
			e.stopPropagation();
			if (!app.arrowMode) {
				// Use observer pattern instead of direct call
				//app.propertiesPanelSubject.notifyPropertiesPanelRequested(this);
			}
		});

		app.canvas.appendChild(this.group); // ✅ now correctly adding full group
		app.placedComponents.push(this);
		app.runButton.disabled = false;

		this.x = x;
		this.y = y;
	}

	initDrag() {
		let offsetX, offsetY;

		const onMouseMove = (e) => {
			const pt = this.app.canvas.createSVGPoint();
			pt.x = e.clientX;
			pt.y = e.clientY;
			const svgP = pt.matrixTransform(this.app.canvas.getScreenCTM().inverse());

			const newX = svgP.x - offsetX;
			const newY = svgP.y - offsetY;

			this.group.setAttribute('transform', `translate(${newX}, ${newY})`);

			this.x = newX;
			this.y = newY;

			this.app.updateConnectionsFor(this);
		};

		const onMouseUp = () => {
			window.removeEventListener('mousemove', onMouseMove);
			window.removeEventListener('mouseup', onMouseUp);
		};

		this.group.addEventListener('mousedown', (e) => {
			e.preventDefault();
			const pt = this.app.canvas.createSVGPoint();
			pt.x = e.clientX;
			pt.y = e.clientY;
			const svgP = pt.matrixTransform(this.app.canvas.getScreenCTM().inverse());

			const ctm = this.group.getCTM();
			offsetX = svgP.x - ctm.e;
			offsetY = svgP.y - ctm.f;

			window.addEventListener('mousemove', onMouseMove);
			window.addEventListener('mouseup', onMouseUp);
		});
	}

	updateLabel(newLabel) {
		this.props.label = newLabel;
		this.text.textContent = newLabel;
		const textWidth = this.text.getBBox().width;
		const padding = 20;
		const finalWidth = textWidth + padding;

		this.group.querySelector('rect').setAttribute('width', finalWidth);
		this.text.setAttribute('x', parseFloat(this.group.querySelector('rect').getAttribute('x')) + finalWidth / 2);

	}

	getCenter() {
		const bbox = this.group.getBBox();
		const ctm = this.group.getCTM();
		const x = ctm.e + bbox.x + bbox.width / 2;
		const y = ctm.f + bbox.y + bbox.height / 2;
		return { x, y };
	}

	select() {
		// Use observer to clear previous selection and select this node
		if (this.app.selectedNode && this.app.selectedNode !== this) {
			this.app.nodeSelectionSubject.notifyNodeDeselected(this.app.selectedNode);
		}
		this.group.classList.add('selected');
		this.app.selectedNode = this;
	}

	deselect() {
		this.group.classList.remove('selected');
		if (this.app.selectedNode === this) {
			this.app.selectedNode = null;
		}
	}

	getConnectionPointToward(otherNode) {
		const bbox = this.group.getBBox();
		const ctm = this.group.getCTM();

		const centerX = ctm.e + bbox.x + bbox.width / 2;
		const centerY = ctm.f + bbox.y + bbox.height / 2;

		const otherCenter = otherNode.getCenter();

		let edgeX = centerX;
		let edgeY = centerY;

		const dx = otherCenter.x - centerX;
		const dy = otherCenter.y - centerY;

		if (Math.abs(dx) > Math.abs(dy)) {
			edgeX += dx > 0 ? bbox.width / 2 : -bbox.width / 2;
		} else {
			edgeY += dy > 0 ? bbox.height / 2 : -bbox.height / 2;
		}

		return { x: edgeX, y: edgeY };
	}
}
