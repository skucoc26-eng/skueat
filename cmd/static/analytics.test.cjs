const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, 'analytics.js'), 'utf8');

function app(optOut = false, fetchImpl = async () => ({})) {
    let ready;
    const storage = new Map(optOut ? [['analytics-opt-out', '1']] : []);
    const checkbox = { addEventListener(type, fn) { this.change = fn; } };
    const calls = [];
    const context = {
        window: { location: { search: '?utm_source=club' } },
        document: { addEventListener(type, fn) { ready = fn; }, getElementById() { return checkbox; } },
        localStorage: { getItem(k) { return storage.get(k); }, setItem(k, v) { storage.set(k, v); } },
        AbortController, URLSearchParams, setTimeout, clearTimeout,
        fetch(...args) { calls.push(args); return fetchImpl(...args); },
    };
    vm.createContext(context); vm.runInContext(source, context); ready();
    return { context, calls, checkbox };
}
const tick = () => new Promise(resolve => setImmediate(resolve));

test('pageview and restaurant action are serialized to preserve visitor identity', async () => {
    let release;
    const a = app(false, () => new Promise(resolve => { release = resolve; }));
    a.context.window.skueatTrack('detail', { restaurant_id: 227 });
    await tick(); assert.equal(a.calls.length, 1);
    assert.deepEqual(JSON.parse(a.calls[0][1].body), { type: 'page_view', source: 'club' });
    release({}); await tick(); assert.equal(a.calls.length, 2);
    assert.deepEqual(JSON.parse(a.calls[1][1].body), { type: 'detail', restaurant_id: 227 });
    release({}); await tick();
});
test('opt-out suppresses events and can be changed in settings', async () => {
    const a = app(true); a.context.window.skueatTrack('search', { query: '후로' });
    await tick(); assert.equal(a.calls.length, 0); assert.equal(a.checkbox.checked, true);
    a.checkbox.checked = false; a.checkbox.change(); await tick(); assert.equal(a.calls.length, 1);
});
test('failed measurement does not break later actions', async () => {
    const a = app(false, async () => { throw new Error('offline'); });
    await tick(); assert.doesNotThrow(() => a.context.window.skueatTrack('directions', { restaurant_id: 1 }));
    await tick(); assert.equal(a.calls.length, 2);
});
