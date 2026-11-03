const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

function admin() {
    const elements = new Map();
    const node = () => ({ children: [], value: '', append(...items) { this.children.push(...items); }, replaceChildren(...items) { this.children = items; }, setAttribute() {}, listeners: {}, addEventListener(type, fn) { this.listeners[type] = fn; } });
    const get = id => { if (!elements.has(id)) elements.set(id, node()); return elements.get(id); };
    get('review-status').value = 'all';
    const requests = [];
    const context = {
        document: { addEventListener() {}, getElementById: get, createElement: node },
        URLSearchParams,
        window: { confirm() { return true; } },
        fetch(url, options) { return new Promise(resolve => requests.push({ url, options, resolve })); },
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

function userReply(a, index, nickname, role = 'user') {
    a.requests[index].resolve({ ok: true, status: 200, json: async () => ({ total: 1, items: [{ id: 1, nickname, role, review_count: 2, first_login_at: '2026-11-03T00:00:00Z', last_login_at: '2026-11-03T01:00:00Z' }] }) });
}
test('latest user search wins and logout discards pending private rows', async () => {
    const a = admin(); vm.runInContext('isOwner = true', a.context);
    const first = vm.runInContext('loadUsers()', a.context);
    a.get('user-search').value = 'new';
    const second = vm.runInContext('loadUsers()', a.context);
    userReply(a, 1, 'new'); await second; userReply(a, 0, 'old'); await first;
    assert.match(a.get('users').children[0].children[0].textContent, /new/);
    const pending = vm.runInContext('loadUsers()', a.context);
    vm.runInContext('lock()', a.context); userReply(a, 2, 'private'); await pending;
    assert.equal(a.get('users').children.length, 0);
    assert.equal(a.get('user-management').hidden, true);
});
test('role button uses internal ID and CSRF; nickname is text', async () => {
    const a = admin(); vm.runInContext("isOwner = true; csrf = 'token'", a.context);
    const first = vm.runInContext('loadUsers()', a.context);
    userReply(a, 0, '<img src=x onerror=alert(1)>'); await first;
    const card = a.get('users').children[0]; assert.equal('innerHTML' in card, false);
    assert.match(card.children[0].textContent, /<img/);
    const button = card.children.at(-1); const action = button.listeners.click();
    assert.equal(a.requests[1].url, '/api/admin/users/1/role');
    assert.deepEqual(JSON.parse(a.requests[1].options.body), { role: 'operator' });
    assert.equal(a.requests[1].options.headers['X-CSRF-Token'], 'token');
    a.requests[1].resolve({status:204});
    await new Promise(resolve => setImmediate(resolve));
    userReply(a, 2, 'new operator', 'operator');
    a.requests[3].resolve({ok:true,status:200,json:async()=>[]}); await action;
    assert.equal(a.get('users').children[0].children.at(-1).textContent, '운영자 권한 회수');
});
test('operator refresh hides user management and never fetches its APIs', async () => {
    const a = admin(); a.context.calls = [];
    vm.runInContext("loadStats = async () => calls.push('stats'); loadReviews = async () => calls.push('reviews'); loadAudit = async () => calls.push('audit'); loadUsers = async () => calls.push('users'); loadRoleAudit = async () => calls.push('roles'); isOwner = false; unlock()", a.context);
    await vm.runInContext('refresh()', a.context);
    assert.deepEqual(Array.from(a.context.calls), ['stats','reviews','audit']);
    assert.equal(a.get('user-management').hidden, true);
});
