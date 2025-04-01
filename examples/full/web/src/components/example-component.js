class ExampleComponent extends HTMLElement {
    constructor() {
        super();
        this.attachShadow({ mode: 'open' });
        this.shadowRoot.innerHTML = `
            <style>
                :host {
                    display: block;
                    padding: 1em;
                    border: 1px solid #ccc;
                    border-radius: 4px;
                    margin-bottom: 1em;
                }
                .container {
                    text-align: center;
                }
                button {
                    padding: 0.5em 1em;
                    background-color: #4a4a4a;
                    color: white;
                    border: none;
                    border-radius: 4px;
                    cursor: pointer;
                }
            </style>
            <div class="container">
                <h2>Example Web Component</h2>
                <button id="btn">Click Me</button>
                <p id="output"></p>
            </div>
        `;
        
        this.btn = this.shadowRoot.getElementById('btn');
        this.output = this.shadowRoot.getElementById('output');
        
        this.btn.addEventListener('click', () => this.handleClick());
    }
    
    handleClick() {
        this.output.textContent = `Button clicked at ${new Date().toLocaleTimeString()}`;
    }
}

customElements.define('example-component', ExampleComponent);
