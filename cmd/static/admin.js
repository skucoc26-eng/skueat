'use strict';
let csrf = '';
let reviewPage = 1;
let statsRequest = 0;
const $ = id => document.getElementById(id);
const number = value => Number(value || 0).toLocaleString('ko-KR');
function message(text, error = false) { $('message').textContent = text; $('message').className = error ? 'error' : ''; }
function lock() { $('dashboard').hidden = true; $('login-panel').hidden = false; $('logout').hidden = true; }
function unlock() { $('dashboard').hidden = false; $('login-panel').hidden = true; $('logout').hidden = false; }
async function api(path, body) {
    const options = { credentials: 'same-origin', cache: 'no-store' };
    if (body !== undefined) { options.method = 'POST'; options.headers = { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }; options.body = JSON.stringify(body); }
    const response = await fetch(`/api/admin/${path}`, options);
    if (response.status === 401) lock();
    if (response.status === 204) return null;
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || '요청을 처리하지 못했습니다.');
    return data;
}
function element(tag, text, className) {
    const node = document.createElement(tag);
    if (text !== undefined) node.textContent = String(text);
    if (className) node.className = className;
    return node;
}
function table(target, headers, rows) {
    target.replaceChildren();
    if (!rows.length) { target.append(element('p', '아직 수집된 데이터가 없습니다.', 'empty')); return; }
    const node = element('table');
    const head = element('tr'); headers.forEach(h => { const th = element('th', h); th.scope = 'col'; head.append(th); });
    const thead = element('thead'); thead.append(head); node.append(thead);
    const tbody = element('tbody'); rows.forEach(row => { const tr = element('tr'); row.forEach(value => tr.append(element('td', value))); tbody.append(tr); });
    node.append(tbody); target.append(node);
}
function ranking(target, rows) {
    target.replaceChildren();
    if (!rows.length) { target.append(element('p', '아직 수집된 데이터가 없습니다.', 'empty')); return; }
    const max = Math.max(1, ...rows.map(r => r.count));
    rows.forEach((row, i) => {
        const line = element('div', undefined, 'rank-row');
        line.append(element('span', `${i + 1}. ${row.name || '직접 방문'}`), element('strong', number(row.count)));
        const bar = element('progress'); bar.max = max; bar.value = row.count; bar.setAttribute('aria-label', `${row.name || '직접 방문'} ${row.count}회`);
        line.append(bar); target.append(line);
    });
}
function dailyChart(rows) {
    const target = $('daily-chart'); target.replaceChildren();
    const ns = 'http://www.w3.org/2000/svg';
    const svg = document.createElementNS(ns, 'svg'); svg.setAttribute('viewBox', '0 0 800 190'); svg.setAttribute('role', 'img'); svg.setAttribute('aria-label', '일별 방문수 막대그래프. 아래 일별 숫자 보기에서 정확한 수치를 확인하세요.');
    const max = Math.max(1, ...rows.map(r => r.page_views)); const step = 800 / rows.length;
    rows.forEach((r, i) => {
        const rect = document.createElementNS(ns, 'rect'); const height = r.page_views / max * 145;
        Object.entries({ x: i * step + 2, y: 155 - height, width: Math.max(2, step - 4), height, rx: 2 }).forEach(([k, v]) => rect.setAttribute(k, v));
        const title = document.createElementNS(ns, 'title'); title.textContent = `${r.day}: 방문 ${r.page_views}, 브라우저 ${r.visitors}`; rect.append(title); svg.append(rect);
        if (i === 0 || i === rows.length - 1 || i % Math.ceil(rows.length / 7) === 0) {
            const label = document.createElementNS(ns, 'text'); label.setAttribute('x', i * step + 2); label.setAttribute('y', 180); label.textContent = r.day.slice(5); svg.append(label);
        }
    });
    target.append(svg);
    table($('daily-table'), ['날짜 (KST)', '방문수', '익명 브라우저'], rows.map(r => [r.day, number(r.page_views), number(r.visitors)]));
}
async function loadStats() {
    const request = ++statsRequest;
    const data = await api(`stats?days=${encodeURIComponent($('days').value)}`);
    if (request !== statsRequest) return;
    $('tracking-note').textContent = data.tracking_since ? `수집된 기록 시작: ${new Date(data.tracking_since).toLocaleString('ko-KR')} · 조회 ${data.start}부터 (KST)` : '배포 이후 사용 통계가 수집됩니다. 과거 수치는 소급 집계하지 않습니다.';
    const metrics = [['방문수', data.page_views], ['익명 브라우저', data.visitors], ['검색', data.searches], ['상세조회', data.details], ['길찾기 클릭', data.directions], ['룰렛 선정', data.roulette], ['기간 내 등록 리뷰', data.reviews], ['현재 숨김 리뷰', data.hidden_reviews]];
    $('metrics').replaceChildren(); metrics.forEach(([label, value]) => { const card = element('article'); card.append(element('p', label), element('strong', number(value))); $('metrics').append(card); });
    dailyChart(data.daily); ranking($('queries'), data.queries); ranking($('categories'), data.categories); ranking($('sources'), data.sources);
    table($('restaurants'), ['음식점', '상세조회', '길찾기', '룰렛 선정'], data.restaurants.map(r => [r.title, number(r.details), number(r.directions), number(r.roulette)]));
}
async function loadReviews() {
    const params = new URLSearchParams({ page: reviewPage, status: $('review-status').value, search: $('review-search').value });
    const data = await api(`reviews?${params}`); const target = $('reviews'); target.replaceChildren();
    if (!data.items.length) target.append(element('p', '조건에 맞는 리뷰가 없습니다.', 'empty'));
    data.items.forEach(review => {
        const card = element('article', undefined, `review ${review.hidden ? 'is-hidden' : ''}`);
        card.append(element('h3', `${review.restaurant} · ${review.score}점`), element('p', `${review.hidden ? '숨김' : '공개'} · ${review.author_name || '익명'} · ${new Date(review.created_at).toLocaleString('ko-KR')}`, 'note'), element('p', review.comment || '(별점만 등록)', 'comment'));
        if (review.moderation_reason) card.append(element('p', `최근 관리 사유: ${review.moderation_reason}`, 'note'));
        const form = element('form', undefined, 'moderation-form'); const reason = element('input'); reason.required = true; reason.maxLength = 200; reason.placeholder = review.hidden ? '복구 사유 (필수)' : '숨김 사유 (필수)'; reason.setAttribute('aria-label', `리뷰 ${review.id} 관리 사유`);
        const button = element('button', review.hidden ? '리뷰 복구' : '리뷰 숨김'); button.type = 'submit'; if (!review.hidden) button.className = 'danger'; form.append(reason, button);
        form.addEventListener('submit', async event => {
            event.preventDefault(); button.disabled = true;
            try { await api(`reviews/${review.id}/moderation`, { hidden: !review.hidden, reason: reason.value }); await Promise.all([loadReviews(), loadStats(), loadAudit()]); message('리뷰 상태와 평점 집계를 갱신했습니다.'); }
            catch (error) { message(error.message, true); } finally { button.disabled = false; }
        }); card.append(form); target.append(card);
    });
    $('review-page').textContent = `${reviewPage} / ${Math.max(1, Math.ceil(data.total / 50))} 페이지 · ${number(data.total)}건`;
    $('previous').disabled = reviewPage <= 1; $('next').disabled = reviewPage * 50 >= data.total;
}
async function loadAudit() {
    const rows = await api('audit');
    table($('audit'), ['시각', '관리자', '리뷰 ID', '작업', '사유'], rows.map(r => [new Date(r.created_at).toLocaleString('ko-KR'), r.admin, r.review_id, r.hidden ? '숨김' : '복구', r.reason]));
}
async function refresh() { try { await Promise.all([loadStats(), loadReviews(), loadAudit()]); message('최신 데이터를 불러왔습니다.'); } catch (error) { message(error.message, true); } }
async function initialize() {
    try {
        const session = await api('session'); csrf = session.csrf; $('username').value = session.username;
        if (!session.enabled) message('관리자 계정 설정이 필요합니다.', true);
        if (session.authenticated) { unlock(); await refresh(); } else lock();
    } catch (error) { message(error.message, true); }
}
document.addEventListener('DOMContentLoaded', () => {
    $('login-form').addEventListener('submit', async event => {
        event.preventDefault(); const button = event.currentTarget.querySelector('button'); button.disabled = true;
        try { const result = await api('login', { username: $('username').value, password: $('password').value }); csrf = result.csrf; $('password').value = ''; unlock(); await refresh(); }
        catch (error) { message(error.message, true); await initialize(); } finally { button.disabled = false; }
    });
    $('logout').addEventListener('click', async () => { try { await api('logout', {}); lock(); await initialize(); message('로그아웃했습니다.'); } catch (error) { message(error.message, true); } });
    $('days').addEventListener('change', refresh); $('refresh').addEventListener('click', refresh);
    $('review-filter').addEventListener('submit', async e => { e.preventDefault(); reviewPage = 1; try { await loadReviews(); } catch (error) { message(error.message, true); } });
    $('previous').addEventListener('click', async () => { reviewPage = Math.max(1, reviewPage - 1); try { await loadReviews(); } catch (error) { message(error.message, true); } });
    $('next').addEventListener('click', async () => { reviewPage++; try { await loadReviews(); } catch (error) { message(error.message, true); } });
    initialize();
});
