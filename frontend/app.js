const routes = {};

function registerPage(hash, renderFn) {
    routes[hash] = renderFn;
}

function navigate() {
    const hash = window.location.hash || '#dashboard';
    const app = document.getElementById('app');
    document.querySelectorAll('.nav-link').forEach(l => {
        l.classList.toggle('active', l.getAttribute('href') === hash);
    });
    const render = routes[hash];
    if (render) {
        app.innerHTML = '';
        render(app);
    }
}

function showToast(message, type) {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast toast-' + (type || 'info');
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => toast.remove(), 4000);
}

window.addEventListener('hashchange', navigate);
window.addEventListener('DOMContentLoaded', function() {
    navigate();
    // Navigate to learning page when backend requests it
    window.runtime.EventsOn('navigate:learning', function() {
        window.location.hash = '#learning';
    });
    // Display version in navbar
    (async () => {
        try {
            const version = await window.go.main.App.GetVersion();
            const el = document.getElementById('app-version');
            if (el && version) el.textContent = 'v' + version;
        } catch(e) {}
    })();
});
