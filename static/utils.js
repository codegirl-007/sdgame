let nodeIdCounter = 1;

/**
 * Generates a unique node ID like "node-1", "node-2", etc.
 * @returns {string}
 */
export function generateNodeId() {
    return `node-${nodeIdCounter++}`;
}

/**
 * Creates an SVG element with the given tag and attributes.
 * @param {string} tag - The SVG tag name (e.g., 'rect', 'line').
 * @param {Object} attrs - An object of attribute key-value pairs.
 * @returns {SVGElement}
 */
export function createSVGElement(tag, attrs) {
    const elem = document.createElementNS('http://www.w3.org/2000/svg', tag);
    for (let key in attrs) {
        elem.setAttribute(key, attrs[key]);
    }
    return elem;
}
