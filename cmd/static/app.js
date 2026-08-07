let map;
let mapMarkers = [];
let userCoords = null;
let userLocOverlay = null;
let userCircle = null;
let isGpsActive = false;
let currentRestaurants = []; // 룰렛/랜덤 매칭에 사용될 현재 필터링된 맛집 목록 캐시
let isRouletteRunning = false;

// 안양 8동 경계 좌표
const boundaryCoords = [[126.936009, 37.38326], [126.936242, 37.382663], [126.936444, 37.382211], [126.936508, 37.382068], [126.936646, 37.381803], [126.936743, 37.381642], [126.936765, 37.381605], [126.93691, 37.381366], [126.937138, 37.381042], [126.937483, 37.380612], [126.937944, 37.380137], [126.938376, 37.379733], [126.938679, 37.379465], [126.938863, 37.379302], [126.939684, 37.378558], [126.940026, 37.37823], [126.940195, 37.378053], [126.940214, 37.378032], [126.940296, 37.377945], [126.940511, 37.377704], [126.940739, 37.377422], [126.940785, 37.37736], [126.940877, 37.377235], [126.940956, 37.377128], [126.941185, 37.376762], [126.940555, 37.376242], [126.940511, 37.376205], [126.940318, 37.376048], [126.940076, 37.375851], [126.939165, 37.376011], [126.937373, 37.376555], [126.934893, 37.376996], [126.932765, 37.37747], [126.930979, 37.377528], [126.926947, 37.377638], [126.924137, 37.377312], [126.923923, 37.377256], [126.921417, 37.376518], [126.919899, 37.376564], [126.919178, 37.376933], [126.91921, 37.377178], [126.919285, 37.377544], [126.919353, 37.377714], [126.919431, 37.377818], [126.920582, 37.379335], [126.921934, 37.380899], [126.924169, 37.383026], [126.925436, 37.384031], [126.925688, 37.384199], [126.925741, 37.384224], [126.92579, 37.384248], [126.932237, 37.385795], [126.93377, 37.383967], [126.933825, 37.383896], [126.936009, 37.383261]];

function initMap() {
    const container = document.getElementById('map');
    const options = { center: new kakao.maps.LatLng(37.382, 126.931), level: 3 };
    map = new kakao.maps.Map(container, options);

    const path = boundaryCoords.map(c => new kakao.maps.LatLng(c[1], c[0]));
    new kakao.maps.Polygon({
        path: path, strokeWeight: 3, strokeColor: '#FF0000', strokeOpacity: 0.6,
        fillColor: '#FF0000', fillOpacity: 0.05
    }).setMap(map);
}

// 🌓 다크 모드 초기화 및 제어 기능
function initTheme() {
    const themeBtn = document.getElementById('btn-theme');
    if (!themeBtn) return;

    const savedTheme = localStorage.getItem('theme');
    const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

    if (savedTheme === 'dark' || (!savedTheme && systemPrefersDark)) {
        document.body.classList.add('dark-mode');
        themeBtn.innerHTML = '☀️';
    } else {
        document.body.classList.remove('dark-mode');
        themeBtn.innerHTML = '🌙';
    }

    themeBtn.addEventListener('click', () => {
        document.body.classList.toggle('dark-mode');
        const isDark = document.body.classList.contains('dark-mode');
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
        themeBtn.innerHTML = isDark ? '☀️' : '🌙';
    });
}

// 두 좌표 사이의 도보/직선 거리 계산 (Haversine 공식)
function getDistance(lat1, lon1, lat2, lon2) {
    const R = 6371000; // 지구 반지름 (m)
    const dLat = (lat2 - lat1) * Math.PI / 180;
    const dLon = (lon2 - lon1) * Math.PI / 180;
    const a = 
        Math.sin(dLat/2) * Math.sin(dLat/2) +
        Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) * 
        Math.sin(dLon/2) * Math.sin(dLon/2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
    return R * c; // 미터 단위 거리
}

// GPS 위치 요청 함수
function requestUserLocation() {
    if (!navigator.geolocation) {
        alert("이 브라우저에서는 위치 서비스를 지원하지 않습니다.");
        return;
    }

    const gpsBtn = document.getElementById('btn-gps');
    if (gpsBtn) gpsBtn.innerHTML = "🌀"; // 로딩 표시

    navigator.geolocation.getCurrentPosition(
        (position) => {
            const lat = position.coords.latitude;
            const lng = position.coords.longitude;
            userCoords = { lat, lng };
            isGpsActive = true;

            if (gpsBtn) {
                gpsBtn.classList.add('active');
                gpsBtn.innerHTML = "📍";
            }

            const locPosition = new kakao.maps.LatLng(lat, lng);

            // 내 위치 마커 그리기
            if (userLocOverlay) userLocOverlay.setMap(null);
            
            const pulseEl = document.createElement('div');
            pulseEl.style.width = '16px';
            pulseEl.style.height = '16px';
            pulseEl.style.borderRadius = '50%';
            pulseEl.style.backgroundColor = '#007aff';
            pulseEl.style.border = '3px solid #fff';
            pulseEl.style.boxShadow = '0 0 10px rgba(0,122,255,0.6)';
            
            userLocOverlay = new kakao.maps.CustomOverlay({
                position: locPosition,
                content: pulseEl,
                xAnchor: 0.5,
                yAnchor: 0.5
            });
            userLocOverlay.setMap(map);

            // 정확도 원 그리기
            if (userCircle) userCircle.setMap(null);
            userCircle = new kakao.maps.Circle({
                center: locPosition,
                radius: 80,
                strokeWeight: 0,
                fillColor: '#007aff',
                fillOpacity: 0.12
            });
            userCircle.setMap(map);

            map.panTo(locPosition);

            // 거리 순 정렬을 위해 리스트 새로고침
            applySearch();
        },
        (error) => {
            console.error("위치 획득 실패", error);
            alert("위치 정보를 가져올 수 없습니다. GPS 권한을 확인해 주세요.");
            isGpsActive = false;
            if (gpsBtn) {
                gpsBtn.classList.remove('active');
                gpsBtn.innerHTML = "📍";
            }
        },
        { enableHighAccuracy: true, timeout: 5000, maximumAge: 0 }
    );
}

async function fetchData(category = 'all', search = '') {
    const url = `/api/restaurants?category=${category}&search=${search}`;
    try {
        const response = await fetch(url);
        const data = await response.json();
        
        // 내 위치 정보가 있으면 거리 계산 및 소팅 적용
        if (userCoords) {
            data.forEach(item => {
                item.distance = getDistance(userCoords.lat, userCoords.lng, item.y, item.x);
            });
            data.sort((a, b) => a.distance - b.distance);
        }

        currentRestaurants = data; // 룰렛용 현재 맛집 목록 캐싱
        renderList(data);
    } catch (e) { console.error("로드 실패", e); }
}

function renderList(data) {
    const container = document.getElementById('res-list');
    document.getElementById('list-count').textContent = `주변 맛집 ${data.length}곳`;
    container.innerHTML = '';

    // 기존 맵 마커 지우기
    mapMarkers.forEach(marker => marker.setMap(null));
    mapMarkers = [];

    data.forEach(item => {
        const card = document.createElement('div');
        card.className = 'res-card';
        card.dataset.id = item.ID;
        
        const avg = item.avg_rating || 0;
        const count = item.rating_count || 0;

        // 음식 카테고리 태그와 거리 텍스트 세팅
        let distanceHtml = '';
        if (item.distance !== undefined) {
            const distText = item.distance < 1000 ? `${Math.round(item.distance)}m` : `${(item.distance / 1000).toFixed(1)}km`;
            distanceHtml = `<span class="res-distance">🚶 ${distText}</span>`;
        }

        card.innerHTML = `
            <div class="res-card-header">
                <span class="res-food-tag">#${item.food}</span>
                ${distanceHtml}
            </div>
            <a class="res-title" href="${item.url}" target="_blank" onclick="event.stopPropagation()">${item.title}</a>
            <div class="rating-box">
                <span class="star-display filled">★</span>
                <span class="rating-score">${avg.toFixed(1)}</span>
                <span class="rating-count">(${count})</span>
            </div>
            <div class="res-addr">${item.addr}</div>
        `;
        
        card.onclick = () => focusOn(item, card);
        container.appendChild(card);

        // 지도 위의 커스텀 핀 오버레이 마커 만들기
        const pos = new kakao.maps.LatLng(item.y, item.x);
        const markerEl = document.createElement('div');
        
        let pinClass = 'custom-overlay-pin';
        let emoji = '📍';
        if (item.food.includes('카페')) { pinClass += ' cafe'; emoji = '☕'; }
        else if (item.food.includes('국수')) { pinClass += ' noodle'; emoji = '🍜'; }
        else if (item.food.includes('중식')) { pinClass += ' chinese'; emoji = '🇨🇳'; }
        else if (item.food.includes('분식') || item.food.includes('떡볶이')) { pinClass += ' noodle'; emoji = '🍢'; }
        else if (item.food.includes('고기') || item.food.includes('삼겹살') || item.food.includes('닭') || item.food.includes('치킨')) { pinClass += ' meat'; emoji = '🥩'; }
        else if (item.food.includes('술집') || item.food.includes('호프') || item.food.includes('주점')) { pinClass += ' pub'; emoji = '🍺'; }

        markerEl.className = pinClass;
        markerEl.innerHTML = `${emoji} ${item.title}`;

        const overlayMarker = new kakao.maps.CustomOverlay({
            position: pos,
            content: markerEl,
            yAnchor: 1.3
        });
        
        overlayMarker.setMap(map);
        mapMarkers.push(overlayMarker);

        // 데이터와 핀 매핑
        item.overlay = overlayMarker;
        item.markerEl = markerEl;

        // 지도 핀 클릭 이벤트
        markerEl.addEventListener('click', (e) => {
            e.stopPropagation();
            focusOn(item, card);
            card.scrollIntoView({ behavior: 'smooth', block: 'center' });
        });
    });
}

// 특정 식당의 리뷰 불러오기
async function loadReviews(resId, listContainer) {
    try {
        const response = await fetch(`/api/reviews?restaurant_id=${resId}`);
        const reviews = await response.json();
        listContainer.innerHTML = '';

        if (!reviews || reviews.length === 0) {
            listContainer.innerHTML = '<div style="color:#aaa; text-align:center; padding:16px 0; font-size:12px;">첫 번째 한 줄 평을 남겨보세요! ✍️</div>';
            return;
        }

        reviews.forEach(rev => {
            const item = document.createElement('div');
            item.className = 'review-item';
            
            const stars = '★'.repeat(rev.score) + '☆'.repeat(5 - rev.score);
            item.innerHTML = `
                <div class="review-item-header">
                    <span class="review-user">${rev.user_id}</span>
                    <span class="review-stars">${stars}</span>
                </div>
                <div class="review-text">${rev.comment || '별점만 남겼습니다.'}</div>
            `;
            listContainer.appendChild(item);
        });
    } catch (e) {
        listContainer.innerHTML = '<div style="color:red; text-align:center; padding:12px 0; font-size:12px;">한 줄 평 로드 실패</div>';
    }
}

// 한 줄 평 제출 함수
async function submitComment(resId, form, reviewsList, item) {
    const picker = form.querySelector('.star-picker');
    const score = picker.dataset.score || 5;
    const commentInput = form.querySelector('.review-comment-input');
    const comment = commentInput.value.trim();

    const formData = new URLSearchParams();
    formData.append('restaurant_id', resId);
    formData.append('score', score);
    formData.append('comment', comment);

    try {
        const response = await fetch('/api/rate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: formData
        });
        const result = await response.json();
        if (response.ok) {
            alert("한 줄 평이 등록되었습니다! 🎉");
            commentInput.value = '';
            
            // 리뷰 새로고침
            loadReviews(resId, reviewsList);
            
            // 데이터 업데이트
            item.rating_count += 1;
            item.avg_rating = result.new_avg;

            // UI 갱신 (리스트의 평균 평점 및 개수 즉시 변경)
            const ratingScoreEl = form.closest('.res-card').querySelector('.rating-score');
            const ratingCountEl = form.closest('.res-card').querySelector('.rating-count');
            if (ratingScoreEl) ratingScoreEl.textContent = result.new_avg.toFixed(1);
            if (ratingCountEl) ratingCountEl.textContent = `(${item.rating_count})`;
            
            // 한 줄 평 목록 제목도 갱신
            const titleEl = form.closest('.reviews-section').querySelector('.reviews-title');
            if (titleEl) titleEl.innerHTML = `💬 한 줄 평 목록 (${result.new_avg.toFixed(1)} / 5.0)`;
        } else {
            alert(result.error || "평가 등록 실패");
        }
    } catch (e) {
        alert("네트워크 오류가 발생했습니다.");
    }
}

// 평점 입력 폼 스타 토글러
function setFormScore(starEl, score) {
    const picker = starEl.parentElement;
    picker.dataset.score = score;
    const stars = picker.querySelectorAll('.star-picker-star');
    stars.forEach(star => {
        const val = parseInt(star.dataset.score);
        if (val <= score) {
            star.classList.add('active');
        } else {
            star.classList.remove('active');
        }
    });
}

function focusOn(item, cardElement) {
    // 1. 다른 모든 카드의 활성 상태 제거 및 리뷰 목록 닫기
    document.querySelectorAll('.res-card').forEach(c => {
        c.classList.remove('active');
        const existingReviews = c.querySelector('.reviews-section');
        if (existingReviews) existingReviews.remove();
    });

    if (!cardElement) return;
    cardElement.classList.add('active');

    // 2. 지도 위 마커 활성화 상태 토글
    mapMarkers.forEach(marker => {
        const el = marker.getContent();
        if (el && el.classList) el.classList.remove('active');
        marker.setZIndex(1);
    });

    if (item.overlay) {
        item.overlay.setZIndex(999);
        const el = item.overlay.getContent();
        if (el && el.classList) el.classList.add('active');
    }

    const pos = new kakao.maps.LatLng(item.y, item.x);
    map.setLevel(2); // 지도를 디테일하게 줌인 (기본 3에서 2로 줌인 최적화)
    map.panTo(pos);

    // 3. 한 줄 평 레이아웃 생성 및 마운트
    const reviewsSec = document.createElement('div');
    reviewsSec.className = 'reviews-section';
    reviewsSec.onclick = (e) => e.stopPropagation(); // 카드 클릭이 닫기 처리되지 않게 방지

    const reviewsTitle = document.createElement('div');
    reviewsTitle.className = 'reviews-title';
    reviewsTitle.innerHTML = `💬 한 줄 평 목록 (${(item.avg_rating || 0).toFixed(1)} / 5.0)`;
    reviewsSec.appendChild(reviewsTitle);

    const reviewsList = document.createElement('div');
    reviewsList.className = 'reviews-list';
    reviewsList.innerHTML = '<div style="text-align:center; padding:12px; color:#999; font-size:12px;">한 줄 평 불러오는 중...</div>';
    reviewsSec.appendChild(reviewsList);

    // 로그인 상태 유무에 따른 작성 폼 노출
    if (IS_LOGGED_IN) {
        const form = document.createElement('form');
        form.className = 'review-form';
        form.onsubmit = (e) => {
            e.preventDefault();
            submitComment(item.ID, form, reviewsList, item);
        };
        form.innerHTML = `
            <div class="review-form-header">
                <span>한 줄 평 쓰기</span>
                <div class="star-picker" data-score="5">
                    <span class="star-picker-star active" data-score="1" onclick="setFormScore(this, 1)">★</span>
                    <span class="star-picker-star active" data-score="2" onclick="setFormScore(this, 2)">★</span>
                    <span class="star-picker-star active" data-score="3" onclick="setFormScore(this, 3)">★</span>
                    <span class="star-picker-star active" data-score="4" onclick="setFormScore(this, 4)">★</span>
                    <span class="star-picker-star active" data-score="5" onclick="setFormScore(this, 5)">★</span>
                </div>
            </div>
            <div class="review-input-row">
                <input type="text" placeholder="한 줄 평을 남겨보세요! (최대 50자)" class="review-comment-input" required maxlength="50">
                <button type="submit" class="review-submit-btn">등록</button>
            </div>
        `;
        reviewsSec.appendChild(form);
    } else {
        const loginMsg = document.createElement('div');
        loginMsg.className = 'review-login-msg';
        loginMsg.innerHTML = `평가 및 한 줄 평 작성을 위해 <a class="review-login-link" href="/login/kakao">💬 카카오 로그인</a>이 필요합니다.`;
        reviewsSec.appendChild(loginMsg);
    }

    cardElement.appendChild(reviewsSec);

    // 리뷰 실시간 로드
    loadReviews(item.ID, reviewsList);

    // 모바일(화면 너비 768px 이하)일 때 바텀시트를 충분히 올려주기
    if (window.innerWidth <= 768) {
        const sheet = document.getElementById('bottom-sheet');
        if (sheet) {
            const mainContent = document.querySelector('.main-content');
            const mainHeight = mainContent.getBoundingClientRect().height;
            const currentHeight = sheet.getBoundingClientRect().height;
            if (currentHeight < mainHeight * 0.4) {
                sheet.style.transition = 'height 0.3s ease-out';
                sheet.style.height = '45%';
            }
        }
    }
}

function applyFilter(category, btn) {
    document.querySelectorAll('.cat-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    fetchData(category, document.getElementById('search-input').value);
}

function applySearch() {
    const activeBtn = document.querySelector('.cat-btn.active');
    const category = activeBtn ? activeBtn.dataset.category : 'all';
    fetchData(category === 'all' ? 'all' : category, document.getElementById('search-input').value);
}

// 🎲 룰렛 제어 및 초기 설정
function initRoulette() {
    const modal = document.getElementById('roulette-modal');
    const closeBtn = document.getElementById('btn-close-roulette');
    const startBtn = document.getElementById('btn-start-roulette');
    const randomBtn = document.getElementById('btn-random');

    if (!modal || !closeBtn || !startBtn || !randomBtn) return;

    randomBtn.addEventListener('click', () => {
        if (currentRestaurants.length === 0) {
            alert("추천할 음식점이 없습니다! 필터를 확인해 주세요.");
            return;
        }
        modal.classList.add('open');
        resetRoulette();
    });

    closeBtn.addEventListener('click', () => {
        if (isRouletteRunning) return;
        modal.classList.remove('open');
    });

    modal.addEventListener('click', (e) => {
        if (isRouletteRunning) return;
        if (e.target === modal) {
            modal.classList.remove('open');
        }
    });

    startBtn.addEventListener('click', runRouletteAnimation);
}

function resetRoulette() {
    const inner = document.getElementById('roulette-inner');
    const resultBox = document.getElementById('roulette-result');
    const startBtn = document.getElementById('btn-start-roulette');
    
    if (!inner || !resultBox || !startBtn) return;

    inner.style.transition = 'none';
    inner.style.transform = 'translateY(0)';
    inner.innerHTML = '<div class="roulette-item">🎲 룰렛을 돌려보세요! 🎲</div>';
    
    resultBox.classList.remove('show');
    resultBox.innerHTML = '';
    startBtn.disabled = false;
}

function runRouletteAnimation() {
    if (isRouletteRunning || currentRestaurants.length === 0) return;
    
    isRouletteRunning = true;
    
    const startBtn = document.getElementById('btn-start-roulette');
    const inner = document.getElementById('roulette-inner');
    const resultBox = document.getElementById('roulette-result');
    
    if (startBtn) startBtn.disabled = true;
    if (resultBox) resultBox.classList.remove('show');

    // 1. 당첨 맛집 선정
    const winnerIndex = Math.floor(Math.random() * currentRestaurants.length);
    const winner = currentRestaurants[winnerIndex];

    // 2. 흐르는 애니메이션을 위한 아이템 어레이 조립 (30개 배치)
    const totalItems = 30;
    const rouletteItems = [];
    
    for (let i = 0; i < totalItems - 1; i++) {
        const randItem = currentRestaurants[Math.floor(Math.random() * currentRestaurants.length)];
        rouletteItems.push(randItem.title);
    }
    // 마지막에 당첨 항목을 주입하여 이곳에 정확히 멈추게 함
    rouletteItems.push(winner.title);

    inner.innerHTML = rouletteItems.map(title => `<div class="roulette-item">${title}</div>`).join('');
    
    // 강제 리플로우
    inner.style.transition = 'none';
    inner.style.transform = 'translateY(0)';
    inner.offsetHeight;

    // 3. 룰렛 회전 애니메이션 시작 (3.2초간 가속/감속 커브 적용)
    inner.style.transition = 'transform 3.2s cubic-bezier(0.1, 0.8, 0.1, 1)';
    inner.style.transform = `translateY(-${(totalItems - 1) * 100}px)`;

    // 4. 애니메이션 종료 후 처리
    setTimeout(() => {
        isRouletteRunning = false;
        
        if (resultBox) {
            resultBox.innerHTML = `
                <div>🎉 오늘의 추천: <strong>${winner.title}</strong></div>
                <button class="roulette-go-btn" id="btn-go-winner">📍 이 맛집 보러가기</button>
            `;
            resultBox.classList.add('show');
            
            document.getElementById('btn-go-winner').addEventListener('click', () => {
                document.getElementById('roulette-modal').classList.remove('open');
                
                const cards = document.querySelectorAll('.res-card');
                let matchedCard = null;
                cards.forEach(card => {
                    if (parseInt(card.dataset.id) === winner.ID) {
                        matchedCard = card;
                    }
                });

                if (matchedCard) {
                    matchedCard.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    focusOn(winner, matchedCard);
                }
            });
        }
    }, 3300);
}

function initBottomSheet() {
    const sheet = document.getElementById('bottom-sheet');
    const handle = document.getElementById('sheet-handle');
    const mainContent = document.querySelector('.main-content');
    
    if (!sheet || !handle || !mainContent) return;

    let isDragging = false;
    let startY, startHeight;

    handle.addEventListener('touchstart', (e) => {
        isDragging = true;
        startY = e.touches[0].clientY;
        startHeight = sheet.getBoundingClientRect().height;
        sheet.style.transition = 'none'; 
    }, { passive: true });

    document.addEventListener('touchmove', (e) => {
        if (!isDragging) return;
        const deltaY = startY - e.touches[0].clientY;
        let newHeight = startHeight + deltaY;

        const mainHeight = mainContent.getBoundingClientRect().height;
        const minHeight = mainHeight * 0.15;
        const maxHeight = mainHeight * 0.95;
        
        if (newHeight < minHeight) newHeight = minHeight;
        if (newHeight > maxHeight) newHeight = maxHeight;
        
        sheet.style.height = `${newHeight}px`;
    }, { passive: true });

    document.addEventListener('touchend', () => {
        if (!isDragging) return;
        isDragging = false;
        sheet.style.transition = 'height 0.3s ease-out'; 

        const currentHeight = sheet.getBoundingClientRect().height;
        const mainHeight = mainContent.getBoundingClientRect().height;

        if (currentHeight > mainHeight * 0.6) {
            sheet.style.height = '95%';
        } else if (currentHeight < mainHeight * 0.3) {
            sheet.style.height = '15%';
        } else {
            sheet.style.height = '45%';
        }
    });
}

// 초기화 및 이벤트 리스너 등록
document.addEventListener('DOMContentLoaded', () => {
    initTheme();      // 🌓 다크 모드 활성화
    initMap();
    fetchData();
    initBottomSheet();
    initRoulette();   // 🎲 룰렛 모달 활성화

    // 검색어 입력
    document.getElementById('search-input').addEventListener('keyup', (e) => {
        if (e.key === 'Enter') applySearch();
    });

    // GPS 위치 검색 버튼 바인딩
    const gpsBtn = document.getElementById('btn-gps');
    if (gpsBtn) {
        gpsBtn.addEventListener('click', () => {
            if (isGpsActive && userCoords) {
                const pos = new kakao.maps.LatLng(userCoords.lat, userCoords.lng);
                map.panTo(pos);
            } else {
                requestUserLocation();
            }
        });
    }

    // 카테고리 필터
    document.getElementById('category-nav').addEventListener('click', (e) => {
        if (e.target.classList.contains('cat-btn')) {
            applyFilter(e.target.dataset.category, e.target);
        }
    });
});