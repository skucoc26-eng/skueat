import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { searchRestaurants } from '../cmd/static/search.mjs';

const restaurants = [
    { ID: 1, title: '맘스터치', addr: '명학역', food: '치킨', distance: 900 },
    { ID: 2, title: '맘스터치 성결대점', addr: '안양', food: '치킨', distance: 200 },
    { ID: 3, title: '동네치킨', addr: '명학 맘스터치 옆', food: '치킨', distance: 10 },
    { ID: 4, title: '메가 커피', addr: '명학', food: '카페', distance: 20 },
    { ID: 5, title: 'CAFE Hello', addr: '안양', food: '카페', distance: 30 },
];
const ids = (query, category = 'all') => searchRestaurants(restaurants, category, query).map(item => item.ID);

test('spacing, case and canonical Hangul normalization', () => {
    assert.deepEqual(ids(' 메가커피 '), [4]);
    assert.deepEqual(ids('메가 커피'), [4]);
    assert.deepEqual(ids('cafehello'), [5]);
    assert.deepEqual(ids('메가커피'.normalize('NFD')), [4]);
});
test('all words must match, across fields and independent of order', () => {
    assert.deepEqual(ids('명학 치킨'), [3, 1]);
    assert.deepEqual(ids('치킨 명학'), [3, 1]);
    assert.deepEqual(ids('명학 없는단어'), []);
    assert.deepEqual(ids('명학 ㅊㅋ'), [3, 1]);
});
test('initials, category and relevance before distance', () => {
    assert.deepEqual(ids('ㅁㅅㅌㅊ'), [1, 2, 3]);
    assert.deepEqual(ids('맘스터치'), [1, 2, 3]);
    assert.deepEqual(ids('맘스터치', '카페'), []);
    assert.deepEqual(ids(' ', '카페'), [4, 5]);
    assert.deepEqual(ids(''), [3, 4, 5, 2, 1]);
});
test('no GPS preserves input order within a relevance group; input is not mutated', () => {
    const input = restaurants.map(({ distance, ...item }) => item);
    const before = structuredClone(input);
    assert.deepEqual(searchRestaurants(input, 'all', '').map(item => item.ID), [1, 2, 3, 4, 5]);
    assert.deepEqual(input, before);
});

test('live input debounces, waits for composition and keeps keyboard focus', () => {
    const listeners = {};
    const pending = new Map();
    let nextTimer = 0;
    let ready;
    let blurs = 0;
    const calls = [];
    const input = { value: '', blur: () => blurs++, addEventListener: (event, fn) => { listeners[event] = fn; } };
    const context = vm.createContext({
        console, calls,
        document: {
            addEventListener: (_, fn) => { ready = fn; },
            getElementById: id => id === 'search-input' ? input : null,
            querySelector: () => null,
        },
        setTimeout: fn => { pending.set(++nextTimer, fn); return nextTimer; },
        clearTimeout: id => pending.delete(id),
    });
    const code = readFileSync(new URL('../cmd/static/app.js', import.meta.url), 'utf8')
        .replace("import('/static/search.mjs')", 'Promise.resolve({})');
    vm.runInContext(code, context);
    vm.runInContext(`
        initTheme = initMap = initBottomSheet = initCategoryScroll = initRoulette = initSettings = () => {};
        expandBottomSheet = () => {};
        fetchData = (category, query) => calls.push([category, query]);
    `, context);
    ready();
    calls.length = 0;
    input.value = '명'; listeners.input();
    input.value = '명학'; listeners.input();
    assert.equal(pending.size, 1);
    listeners.compositionstart();
    listeners.input();
    assert.equal(pending.size, 0);
    listeners.keydown({ key: 'Enter', isComposing: true });
    assert.equal(calls.length, 0);
    input.value = '명학 ㅊㅋ'; listeners.compositionend();
    [...pending.values()][0]();
    assert.equal(calls[0][1], '명학 ㅊㅋ');
    assert.equal(blurs, 0);
    listeners.keydown({ key: 'Enter', preventDefault() {} });
    assert.equal(blurs, 1);
});
