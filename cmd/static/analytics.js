'use strict';
(() => {
    let queue = Promise.resolve();
    let pending = 0;
    function disabled() { try { return localStorage.getItem('analytics-opt-out') === '1'; } catch { return true; } }
    window.skueatTrack = (type, fields = {}) => {
        if (disabled() || pending >= 30) return;
        pending++;
        // Serialize the first requests so the visitor cookie is set before later events.
        queue = queue.then(async () => {
            if (disabled()) return;
            const controller = new AbortController();
            const timeout = setTimeout(() => controller.abort(), 5000);
            try { await fetch('/api/events', { method: 'POST', credentials: 'same-origin', keepalive: true, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type, ...fields }), signal: controller.signal }); }
            catch { /* Usage measurement must not interrupt the site. */ }
            finally { clearTimeout(timeout); }
        }).catch(() => {}).finally(() => { pending--; });
    };
    document.addEventListener('DOMContentLoaded', () => {
        const checkbox = document.getElementById('analytics-opt-out');
        if (checkbox) {
            checkbox.checked = disabled();
            checkbox.addEventListener('change', () => {
                try { localStorage.setItem('analytics-opt-out', checkbox.checked ? '1' : '0'); } catch { checkbox.checked = true; }
                if (!checkbox.checked) window.skueatTrack('page_view', { source: 'direct' });
            });
        }
        let source = 'direct';
        try { source = new URLSearchParams(window.location.search).get('utm_source')?.slice(0, 60) || 'direct'; } catch {}
        window.skueatTrack('page_view', { source });
    });
})();
