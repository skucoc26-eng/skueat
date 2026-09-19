import { getChoseong } from './vendor/es-hangul-2.4.0.mjs';

const normalize = value => String(value ?? '').normalize('NFC').toLowerCase().replace(/\s+/gu, '');
const isInitials = value => /^[ㄱ-ㅎ]+$/u.test(value);
const matches = (text, term) => (isInitials(term) ? getChoseong(text) : text).includes(term);

export function searchRestaurants(restaurants, category = 'all', query = '') {
    const terms = query.trim().split(/\s+/u).filter(Boolean).map(normalize);
    const whole = normalize(query);
    return restaurants.flatMap(item => {
        const title = normalize(item.title);
        const food = normalize(item.food);
        if (category && category !== 'all' && !food.includes(normalize(category))) return [];
        const fields = [title, normalize(item.addr), food];
        const titleMatch = whole && matches(title, whole);
        if (!titleMatch && !terms.every(term => fields.some(field => matches(field, term)))) return [];
        const comparedTitle = isInitials(whole) ? getChoseong(title) : title;
        const rank = !whole ? 0 : comparedTitle === whole ? 0
            : comparedTitle.startsWith(whole) ? 1 : titleMatch ? 2
            : terms.every(term => matches(title, term)) ? 3 : 4;
        return [{ item, rank }];
    }).sort((a, b) => a.rank - b.rank || (a.item.distance ?? Infinity) - (b.item.distance ?? Infinity))
        .map(result => result.item);
}
