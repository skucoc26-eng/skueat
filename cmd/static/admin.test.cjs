const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

function admin() {
    const elements = new Map();
    const node = () => ({ children: [], value: '', append(...items) { this.children.push(...items); }, replaceChildren(...items) { this.children = items; }, setAttribute() {}, addEventListener() {} });
    const get = id => { if (!elements.has(id)) elements.set(id, node()); return elements.get(id); };
    get('review-status').value = 'all';
    const requests = [];
    const context = {
        document: { addEventListener() {}, getElementById: get, createElement: node },
        URLSearchParams,
        fetch(url) { return new Promise(resolve => requests.push({ url, resolve })); },
    };
    vm.createContext(context);
    vm.runInContext(fs.readFileSync(path.join(__dirname, 'admin.js'), 'utf8'), context);
    const reply = (index, restaurant, page = 1, hidden = false) => requests[index].resolve({
        ok: true, status: 200,
        json: async () => ({ page, total: 51, items: [{ id: index + 1, restaurant, hidden, score: 1, created_at: '2026-11-03T00:00:00Z' }] }),
    });
    return { get, context, reply, requests, load: () => vm.runInContext('loadReviews()', context) };
}

test('older review filter response cannot overwrite the latest result', async () => {
    const a = admin();
    const first = a.load();
    a.get('review-status').value = 'hidden';
    const second = a.load();
    assert.equal(new URL(a.requests[1].url, 'https://example.test').searchParams.get('status'), 'hidden');
    a.reply(1, '숨김 결과', 1, true); await second;
    a.reply(0, '이전 전체 결과'); await first;
    assert.match(a.get('reviews').children[0].children[0].textContent, /숨김 결과/);
});

test('older review page response cannot overwrite rows or pagination', async () => {
    const a = admin();
    const first = a.load();
    vm.runInContext('reviewPage = 2', a.context);
    const second = a.load();
    a.reply(1, '두 번째 페이지', 2); await second;
    a.reply(0, '첫 번째 페이지'); await first;
    assert.match(a.get('reviews').children[0].children[0].textContent, /두 번째 페이지/);
    assert.match(a.get('review-page').textContent, /^2 \/ 2/);
    assert.equal(a.get('previous').disabled, false);
    assert.equal(a.get('next').disabled, true);
});
