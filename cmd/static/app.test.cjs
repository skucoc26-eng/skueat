const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, 'app.js'), 'utf8');

function loadApp(overrides = {}) {
    const warnings = [];
    const context = {
        document: { addEventListener() {}, getElementById() { return {}; } },
        window: {},
        console: { warn(message) { warnings.push(message); } },
        URLSearchParams,
        URL,
        alert() {},
        ...overrides,
    };
    vm.createContext(context);
    vm.runInContext(source, context);
    return { context, warnings };
}

test('review HTML escapes names and comments before inserting them', async () => {
    const inserted = [];
    const container = { innerHTML: '', appendChild(element) { inserted.push(element); } };
    const { context } = loadApp({
        document: {
            addEventListener() {},
            getElementById() { return {}; },
            createElement() { return { innerHTML: '', className: '' }; },
        },
        fetch: async () => ({ json: async () => [{
            score: 5,
            author_name: '<img src=x onerror=alert(1)>',
            comment: '<script>alert(1)</script>',
        }] }),
    });
    context.testContainer = container;
    await vm.runInContext('loadReviews(1, testContainer)', context);
    assert.equal(inserted.length, 1);
    assert.match(inserted[0].innerHTML, /&lt;img src=x onerror=alert\(1\)&gt;/);
    assert.match(inserted[0].innerHTML, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
    assert.doesNotMatch(inserted[0].innerHTML, /<img|<script>/);
});

test('map initialization tolerates a missing Kakao SDK', () => {
    const { context, warnings } = loadApp();
    assert.doesNotThrow(() => vm.runInContext('initMap()', context));
    assert.equal(warnings.length, 1);
});

test('concurrent review submits send one request and restore the button', async () => {
    let resolveRequest;
    let requestCount = 0;
    const button = { disabled: false };
    const form = {
        querySelector(selector) {
            if (selector === '.star-picker') return { dataset: { score: 5 } };
            if (selector === '.review-comment-input') return { value: 'hello' };
            if (selector === '.author-chip.active') return { dataset: { type: 'anon' } };
            if (selector === 'button[type="submit"]') return button;
            return null;
        },
    };
    const { context } = loadApp({
        fetch() {
            requestCount++;
            return new Promise(resolve => { resolveRequest = resolve; });
        },
    });
    context.testForm = form;
    const first = vm.runInContext('submitComment(1, testForm, [], {})', context);
    const second = vm.runInContext('submitComment(1, testForm, [], {})', context);
    assert.equal(requestCount, 1);
    assert.equal(button.disabled, true);
    resolveRequest({ ok: false, json: async () => ({ error: 'test rejection' }) });
    await Promise.all([first, second]);
    assert.equal(button.disabled, false);
});

test('restaurant card escapes data and rejects executable links', () => {
    let card;
    const container = { innerHTML: '', appendChild(element) { card = element; } };
    const { context } = loadApp({
        document: {
            addEventListener() {},
            getElementById(id) {
                if (id === 'res-list') return container;
                if (id === 'list-count') return { textContent: '' };
                return {};
            },
            createElement() {
                return { innerHTML: '', dataset: {}, addEventListener() {}, querySelector() { return null; } };
            },
        },
        kakao: { maps: {
            LatLng: class {},
            CustomOverlay: class { setMap() {} },
        } },
    });
    context.testRestaurant = {
        ID: 1, title: '<img src=x onerror=alert(1)>',
        addr: '<script>alert(1)</script>', food: '<b>양식</b>',
        url: 'javascript:alert(1)', x: 126.93, y: 37.38,
    };
    vm.runInContext('renderList([testRestaurant])', context);
    assert.ok(card);
    assert.match(card.innerHTML, /&lt;img src=x onerror=alert\(1\)&gt;/);
    assert.match(card.innerHTML, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
    assert.match(card.innerHTML, /&lt;b&gt;양식&lt;\/b&gt;/);
    assert.doesNotMatch(card.innerHTML, /<img|<script|<b>|javascript:/i);
});

test('roulette escapes restaurant titles in animation and result', () => {
    const inner = { innerHTML: '', style: {}, offsetHeight: 0 };
    const result = { innerHTML: '', classList: { add() {}, remove() {} } };
    const start = { disabled: false };
    const elements = {
        'roulette-inner': inner,
        'roulette-result': result,
        'btn-start-roulette': start,
        'btn-go-winner': { addEventListener() {} },
    };
    const { context } = loadApp({
        document: {
            addEventListener() {},
            getElementById(id) { return elements[id] || null; },
        },
        setTimeout(callback) { callback(); },
    });
    vm.runInContext('currentRestaurants = [{ ID: 1, title: "<img src=x onerror=alert(1)>", food: "한식" }]; runRouletteAnimation()', context);
    assert.match(inner.innerHTML, /&lt;img src=x onerror=alert\(1\)&gt;/);
    assert.match(result.innerHTML, /&lt;img src=x onerror=alert\(1\)&gt;/);
    assert.doesNotMatch(inner.innerHTML + result.innerHTML, /<img/i);
});
