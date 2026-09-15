let map;
let mapMarkers = [];
let userCoords = null;
let userLocOverlay = null;
let userCircle = null;
let isGpsActive = false;
let currentRestaurants = []; // 룰렛/랜덤 매칭에 사용될 현재 필터링된 맛집 목록 캐시
let isRouletteRunning = false;
let activeTooltipOverlay = null; // 현재 지도 위에 표시 중인 식당 이름 말풍선 오버레이
let showMarkers = true; // 지도 위의 음식점 마커들의 전체 표시 여부

const SUNGKYUL_LAT = 37.382;
const SUNGKYUL_LNG = 126.931;
const MAX_BOUNDS_DISTANCE = 1800; // 성결대 중심 기준 반경 1.8km 이내로 이동 제한

// XSS 방어를 위한 HTML 특수문자 이스케이프 함수
function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function initMap() {
    const container = document.getElementById('map');
    if (!container || !window.kakao || !window.kakao.maps) {
        console.warn("카카오 지도 SDK가 로드되지 않아 지도 초기화를 건너뜁니다.");
        return;
    }

    const sungkyulCenter = new kakao.maps.LatLng(SUNGKYUL_LAT, SUNGKYUL_LNG);
    const options = { center: sungkyulCenter, level: 3 };
    map = new kakao.maps.Map(container, options);

    // 💡 지도 축소/확대 레벨 제한 (캠퍼스 생활권 내로 고정)
    map.setMinLevel(1);
    map.setMaxLevel(5);

    // 💡 지도 이동 범위 제한 (캠퍼스 반경을 지나치게 벗어나면 중심으로 자동 복귀)
    kakao.maps.event.addListener(map, 'dragend', function() {
        const center = map.getCenter();
        const dist = getDistance(SUNGKYUL_LAT, SUNGKYUL_LNG, center.getLat(), center.getLng());
        if (dist > MAX_BOUNDS_DISTANCE) {
            map.panTo(sungkyulCenter);
        }
    });
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

// ⚙️ 서비스 설정(길찾기 앱 선택, 마커 토글) 초기화 기능
function initSettings() {
    const modal = document.getElementById('settings-modal');
    const btnSettings = document.getElementById('btn-settings');
    const btnClose = document.getElementById('btn-close-settings');
    const btnSave = document.getElementById('btn-save-settings');
    const chkMarkers = document.getElementById('chk-markers');

    if (!modal || !btnSettings || !btnClose || !btnSave || !chkMarkers) return;

    // 1. 저장된 값 불러와 UI 매핑
    const savedNaviPref = localStorage.getItem('navi-pref') || 'kakaomap';
    const radio = document.querySelector(`input[name="navi-pref"][value="${savedNaviPref}"]`);
    if (radio) radio.checked = true;

    const savedShowMarkers = localStorage.getItem('show-markers') !== 'false';
    showMarkers = savedShowMarkers;
    chkMarkers.checked = showMarkers;

    // 2. 이벤트 리스너 설정
    btnSettings.addEventListener('click', () => {
        modal.classList.add('open');
    });

    btnClose.addEventListener('click', () => {
        modal.classList.remove('open');
    });

    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.classList.remove('open');
    });

    btnSave.addEventListener('click', () => {
        // 길찾기 기본 앱 저장
        const checkedRadio = document.querySelector('input[name="navi-pref"]:checked');
        if (checkedRadio) {
            localStorage.setItem('navi-pref', checkedRadio.value);
        }

        // 마커 표시 여부 저장
        showMarkers = chkMarkers.checked;
        localStorage.setItem('show-markers', showMarkers ? 'true' : 'false');
        
        // 지도 마커 즉시 업데이트
        toggleMarkersVisibility();

        modal.classList.remove('open');
        alert("설정이 성공적으로 저장되었습니다! ⚙️");
    });
}

// 마커 전체 표시/가리기 스위칭 기능
function toggleMarkersVisibility() {
    mapMarkers.forEach(marker => {
        if (showMarkers) {
            marker.setMap(map);
        } else {
            marker.setMap(null);
        }
    });

    // 마커가 비활성화되면 현재 켜져 있는 툴팁 풍선도 가림
    if (!showMarkers && activeTooltipOverlay) {
        activeTooltipOverlay.setMap(null);
        activeTooltipOverlay = null;
    }
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
    if (activeTooltipOverlay) {
        activeTooltipOverlay.setMap(null);
        activeTooltipOverlay = null;
    }

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
            <div class="res-card-footer">
                <button class="route-link-btn">🚙 길찾기</button>
            </div>
        `;
        
        card.onclick = () => focusOn(item, card);
        container.appendChild(card);

        // 🚙 길찾기 버튼 바인딩
        const routeBtn = card.querySelector('.route-link-btn');
        if (routeBtn) {
            routeBtn.addEventListener('click', (e) => {
                goRoute(item, e);
            });
        }

        // 지도 위의 커스텀 핀 오버레이 마커 만들기
        const pos = new kakao.maps.LatLng(item.y, item.x);
        const markerEl = document.createElement('div');
        
        let pinClass = 'custom-overlay-pin';
        let emoji = '🍽️';

        const f = item.food;
        if (f.includes('한식') || f.includes('백반') || f.includes('찌개') || f.includes('국밥')) {
            pinClass += ' korean'; emoji = '🍚';
        } else if (f.includes('중식') || f.includes('중화요리') || f.includes('짜장') || f.includes('짬뽕') || f.includes('마라')) {
            pinClass += ' chinese'; emoji = '🇨🇳';
        } else if (f.includes('일식') || f.includes('초밥') || f.includes('돈까스') || f.includes('카츠') || f.includes('라멘') || f.includes('스시')) {
            pinClass += ' japanese'; emoji = '🍣';
        } else if (f.includes('양식') || f.includes('파스타') || f.includes('피자') || f.includes('버거') || f.includes('스테이크')) {
            pinClass += ' western'; emoji = '🍝';
        } else if (f.includes('치킨') || f.includes('통닭')) {
            pinClass += ' chicken'; emoji = '🍗';
        } else if (f.includes('분식') || f.includes('떡볶이') || f.includes('김밥') || f.includes('순대')) {
            pinClass += ' snack'; emoji = '🍢';
        } else if (f.includes('고기') || f.includes('삼겹살') || f.includes('갈비') || f.includes('곱창') || f.includes('구이') || f.includes('육류')) {
            pinClass += ' meat'; emoji = '🥩';
        } else if (f.includes('국수') || f.includes('칼국수') || f.includes('냉면') || f.includes('우동') || f.includes('밀면')) {
            pinClass += ' noodle'; emoji = '🍜';
        } else if (f.includes('카페') || f.includes('커피') || f.includes('디저트') || f.includes('베이커리') || f.includes('빵')) {
            pinClass += ' cafe'; emoji = '☕';
        } else if (f.includes('술집') || f.includes('호프') || f.includes('주점') || f.includes('포차') || f.includes('맥주') || f.includes('와인') || f.includes('이자카야')) {
            pinClass += ' pub'; emoji = '🍺';
        } else if (f.includes('아시안') || f.includes('베트남') || f.includes('쌀국수') || f.includes('태국') || f.includes('인도')) {
            pinClass += ' asian'; emoji = '🥟';
        } else {
            pinClass += ' general'; emoji = '🍽️';
        }

        markerEl.className = pinClass;
        markerEl.innerHTML = emoji; // 💡 마커 크기 단순화를 위해 이모지만 노출

        const overlayMarker = new kakao.maps.CustomOverlay({
            position: pos,
            content: markerEl,
            yAnchor: 0.5, // 원형이므로 중앙 정렬
            xAnchor: 0.5
        });
        
        // 설정 상태에 맞추어 맵 마커 표시
        if (showMarkers) {
            overlayMarker.setMap(map);
        }
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

// 특정 맛집의 설정된 전용 길찾기 링크 실행
function goRoute(item, event) {
    if (event) event.stopPropagation();
    
    const pref = localStorage.getItem('navi-pref') || 'kakaomap';
    
    // 식당 이름에서 쉼표(,)를 제거하여 맵 API URL 파싱 에러 방지
    const cleanTitle = item.title.replace(/,/g, ' ');
    const encodedTitle = encodeURIComponent(cleanTitle);
    
    if (pref === 'kakaomap') {
        // 카카오맵 외부 길찾기 웹용 링크 (위도, 경도 순)
        const url = `https://map.kakao.com/link/to/${encodedTitle},${item.y},${item.x}`;
        window.open(url, '_blank');
    } else if (pref === 'kakaonavi') {
        // 카카오내비 앱 실행용 SDK 스키마 (경도, 위도 순)
        const appUrl = `kakaonavi-sdk://navigate?name=${encodedTitle}&coordType=WGS84&x=${item.x}&y=${item.y}`;
        window.location.href = appUrl;
        
        // 모바일 브라우저 락 방지용 딜레이 폴백 (카카오맵 웹 길찾기로 연동)
        setTimeout(() => {
            const fallbackUrl = `https://map.kakao.com/link/to/${encodedTitle},${item.y},${item.x}`;
            window.open(fallbackUrl, '_blank');
        }, 1200);
    } else if (pref === 'navermap') {
        // 네이버 지도 앱/웹 크로스플랫폼 대응
        // 1. 네이버 지도 웹용 v5 경로 (도착 좌표값에 COORD_POI 명시하여 목적지 자동 입력 처리)
        const webUrl = `https://map.naver.com/v5/directions/-/-/${item.x},${item.y},${encodedTitle},,COORD_POI/-/transit`;
        
        // 2. 모바일 브라우저인 경우 네이버 지도 앱 호출 시도
        const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        
        if (isMobile) {
            // 네이버 지도 앱 전용 스키마 실행 (도보 길찾기)
            // 하위 호환성(dlat/dlng/dname) 및 신버전 규격(elat/elng/ename) 파라미터를 동시에 넘겨서 모든 버전의 네이버 지도 앱에 대응합니다.
            const appUrl = `nmap://route/walk?dlat=${item.y}&dlng=${item.x}&dname=${encodedTitle}&elat=${item.y}&elng=${item.x}&ename=${encodedTitle}&appname=skueat`;
            window.location.href = appUrl;
            
            // 앱이 미설치되었거나 구동 실패 시 웹페이지로 대체 로딩
            setTimeout(() => {
                window.open(webUrl, '_blank');
            }, 1200);
        } else {
            // PC 환경은 바로 v5 웹 브라우저 화면 오픈
            window.open(webUrl, '_blank');
        }
    }
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
            
            const scoreVal = Math.max(1, Math.min(5, Number(rev.score) || 5));
            const stars = '★'.repeat(scoreVal) + '☆'.repeat(5 - scoreVal);
            const displayName = escapeHtml(rev.author_name || '익명');
            const commentText = escapeHtml(rev.comment || '별점만 남겼습니다.');
            item.innerHTML = `
                <div class="review-item-header">
                    <span class="review-user">${displayName}</span>
                    <span class="review-stars">${stars}</span>
                </div>
                <div class="review-text">${commentText}</div>
            `;
            listContainer.appendChild(item);
        });
    } catch (e) {
        listContainer.innerHTML = '<div style="color:red; text-align:center; padding:12px 0; font-size:12px;">한 줄 평 로드 실패</div>';
    }
}

// 한 줄 평 제출 중복 방지 플래그
let isSubmittingReview = false;

// 한 줄 평 제출 함수
async function submitComment(resId, form, reviewsList, item) {
    if (isSubmittingReview) return;

    const picker = form.querySelector('.star-picker');
    const score = picker.dataset.score || 5;
    const commentInput = form.querySelector('.review-comment-input');
    const comment = commentInput.value.trim();

    const activeChip = form.querySelector('.author-chip.active');
    const authorType = activeChip ? activeChip.dataset.type : 'masked';
    const customInput = form.querySelector('.custom-nickname-input');
    const customName = customInput ? customInput.value.trim() : '';

    const formData = new URLSearchParams();
    formData.append('restaurant_id', resId);
    formData.append('score', score);
    formData.append('comment', comment);
    formData.append('author_type', authorType);
    formData.append('custom_name', customName);

    isSubmittingReview = true;
    const submitBtn = form.querySelector('button[type="submit"]') || form.querySelector('.submit-btn');
    if (submitBtn) submitBtn.disabled = true;

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
    } finally {
        isSubmittingReview = false;
        if (submitBtn) submitBtn.disabled = false;
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

    // 💡 식당 이름 툴팁 오버레이 띄우기 (마커 온 상태일때만)
    if (activeTooltipOverlay) {
        activeTooltipOverlay.setMap(null);
        activeTooltipOverlay = null;
    }
    
    if (showMarkers) {
        const tooltipEl = document.createElement('div');
        tooltipEl.className = 'pin-name-tooltip';
        tooltipEl.textContent = item.title;

        activeTooltipOverlay = new kakao.maps.CustomOverlay({
            position: pos,
            content: tooltipEl,
            yAnchor: 2.1
        });
        activeTooltipOverlay.setMap(map);
    }

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
            <div class="author-selector-row">
                <div class="author-selector-title">👤 작성자 표기:</div>
                <div class="author-type-chips">
                    <span class="author-chip active" data-type="masked" title="이름 마스킹 (예: 홍*동)">🛡️ 마스킹</span>
                    <span class="author-chip" data-type="anon" title="완전 익명 표기">🎭 익명</span>
                    <span class="author-chip" data-type="custom" title="원하는 닉네임 직접 입력">✏️ 닉네임</span>
                    <span class="author-chip" data-type="real" title="카카오 이름 그대로 표기">💬 카카오 이름</span>
                </div>
                <input type="text" class="custom-nickname-input" placeholder="원하는 닉네임을 입력하세요 (최대 10자)" maxlength="10">
            </div>
            <div class="review-input-row">
                <input type="text" placeholder="한 줄 평을 남겨보세요! (최대 50자)" class="review-comment-input" required maxlength="50">
                <button type="submit" class="review-submit-btn">등록</button>
            </div>
        `;

        // 닉네임 표기 칩 선택 이벤트
        const chips = form.querySelectorAll('.author-chip');
        const customInput = form.querySelector('.custom-nickname-input');
        chips.forEach(chip => {
            chip.addEventListener('click', () => {
                chips.forEach(c => c.classList.remove('active'));
                chip.classList.add('active');
                if (chip.dataset.type === 'custom') {
                    customInput.classList.add('show');
                    customInput.focus();
                } else {
                    customInput.classList.remove('show');
                }
            });
        });

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

// 모바일 하단 네비게이션 바텀시트 확장 제어 함수
function expandBottomSheet(targetPercent = 70) {
    const sheet = document.getElementById('bottom-sheet');
    if (!sheet) return;

    // 미디어쿼리가 모바일 바텀시트 모드인지 계산된 스타일 또는 미디어쿼리로 정확히 확인
    const isMobileSheet = window.innerWidth <= 768 || window.matchMedia('(max-width: 768px)').matches || window.getComputedStyle(sheet).position === 'absolute';
    if (!isMobileSheet) return;

    const mainContent = document.querySelector('.main-content');
    sheet.style.transition = 'height 0.35s cubic-bezier(0.2, 0.9, 0.3, 1)';
    
    if (mainContent) {
        const mainHeight = mainContent.getBoundingClientRect().height;
        const targetPx = Math.round(mainHeight * (targetPercent / 100));
        sheet.style.height = `${targetPx}px`;
    } else {
        sheet.style.height = `${targetPercent}%`;
    }

    // 목록 스크롤을 맨 위로 올려서 검색 결과가 즉시 첫 항목부터 보이게 함
    const listContainer = document.getElementById('res-list');
    if (listContainer) listContainer.scrollTop = 0;
    sheet.scrollTop = 0;
}

function applyFilter(category, btn) {
    document.querySelectorAll('.cat-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');

    // 선택된 카테고리를 화면 안으로 부드럽게 스크롤
    btn.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' });

    // 모바일에서 바텀시트가 너무 작거나 접혀있으면 목록을 볼 수 있도록 55%로 확장
    const sheet = document.getElementById('bottom-sheet');
    if (sheet) {
        const isMobileSheet = window.innerWidth <= 768 || window.matchMedia('(max-width: 768px)').matches || window.getComputedStyle(sheet).position === 'absolute';
        if (isMobileSheet) {
            const mainContent = document.querySelector('.main-content');
            const mainHeight = mainContent ? mainContent.getBoundingClientRect().height : window.innerHeight;
            const currentHeight = sheet.getBoundingClientRect().height;
            if (currentHeight < mainHeight * 0.4) {
                expandBottomSheet(55);
            }
        }
    }

    const searchInput = document.getElementById('search-input');
    fetchData(category, searchInput ? searchInput.value : '');
}

function applySearch() {
    const input = document.getElementById('search-input');
    const activeBtn = document.querySelector('.cat-btn.active');
    const category = activeBtn ? activeBtn.dataset.category : 'all';

    // 모바일 가상 키보드가 검색 목록을 가리지 않도록 포커스 해제
    if (input) input.blur();

    // 💡 모바일에서 검색 시 자동으로 하단 네비게이션 바텀시트를 올려 바로 목록 확인 가능하게 처리
    expandBottomSheet(70);

    const keyword = input ? input.value.trim() : '';
    fetchData(category === 'all' ? 'all' : category, keyword);
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
    
    // 💡 카페/술집 제외 옵션 필터링
    let candidates = [...currentRestaurants];
    const excludeCafe = document.getElementById('chk-roulette-exclude-cafe')?.checked;
    const excludePub = document.getElementById('chk-roulette-exclude-pub')?.checked;

    if (excludeCafe) {
        candidates = candidates.filter(item => !item.food.includes('카페') && !item.food.includes('커피') && !item.food.includes('디저트'));
    }
    if (excludePub) {
        candidates = candidates.filter(item => !item.food.includes('술집') && !item.food.includes('호프') && !item.food.includes('주점') && !item.food.includes('포차'));
    }

    if (candidates.length === 0) {
        alert("선택한 조건에 맞는 음식점이 없습니다! 제외 옵션을 해제해 주세요.");
        return;
    }

    isRouletteRunning = true;
    
    const startBtn = document.getElementById('btn-start-roulette');
    const inner = document.getElementById('roulette-inner');
    const resultBox = document.getElementById('roulette-result');
    
    if (startBtn) startBtn.disabled = true;
    if (resultBox) resultBox.classList.remove('show');

    // 1. 당첨 맛집 선정
    const winnerIndex = Math.floor(Math.random() * candidates.length);
    const winner = candidates[winnerIndex];

    // 2. 흐르는 애니메이션을 위한 아이템 어레이 조립 (30개 배치)
    const totalItems = 30;
    const rouletteItems = [];
    
    for (let i = 0; i < totalItems - 1; i++) {
        const randItem = candidates[Math.floor(Math.random() * candidates.length)];
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
    inner.style.transform = `translateY(-${(totalItems - 1) * 80}px)`;

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

// 🧭 상단 카테고리 바 수평 드래그 스크롤 및 탭 분리 제어
function initCategoryScroll() {
    const nav = document.getElementById('category-nav');
    if (!nav) return;

    let isPointerDown = false;
    let startX = 0;
    let initialScrollLeft = 0;
    let isDragging = false;
    let dragDistance = 0;

    // 1. Pointer Events 지원 (모바일 터치 및 PC 마우스 드래그 공통 지원)
    nav.addEventListener('pointerdown', (e) => {
        if (e.pointerType === 'mouse' && e.button !== 0) return;
        isPointerDown = true;
        isDragging = false;
        dragDistance = 0;
        startX = e.clientX;
        initialScrollLeft = nav.scrollLeft;
    });

    window.addEventListener('pointermove', (e) => {
        if (!isPointerDown) return;
        const deltaX = e.clientX - startX;
        dragDistance = Math.abs(deltaX);
        if (dragDistance > 6) {
            isDragging = true;
            nav.scrollLeft = initialScrollLeft - deltaX;
        }
    });

    const endDrag = () => {
        if (!isPointerDown) return;
        isPointerDown = false;
        setTimeout(() => {
            isDragging = false;
        }, 100);
    };

    window.addEventListener('pointerup', endDrag);
    window.addEventListener('pointercancel', endDrag);

    // 2. 카테고리 버튼 클릭 이벤트 위임
    nav.addEventListener('click', (e) => {
        // 드래그 중이었다면 클릭 무시 (스와이프 스크롤 후 원치 않는 버튼 눌림 방지)
        if (isDragging || dragDistance > 6) {
            e.preventDefault();
            e.stopPropagation();
            return;
        }
        const btn = e.target.closest('.cat-btn');
        if (btn && btn.dataset.category) {
            applyFilter(btn.dataset.category, btn);
        }
    });
}

// 📱 모바일 하단 네비게이션 바텀시트 터치 & 드래그 & 탭 제어
function initBottomSheet() {
    const sheet = document.getElementById('bottom-sheet');
    const handle = document.getElementById('sheet-handle');
    const listInfo = document.getElementById('list-count');
    const mainContent = document.querySelector('.main-content');
    
    if (!sheet || !handle || !mainContent) return;

    let isDragging = false;
    let startY = 0;
    let startHeight = 0;
    let totalDeltaY = 0;
    let capturedElement = null;
    let activePointerId = null;

    function isMobileSheet() {
        return window.innerWidth <= 768 || window.matchMedia('(max-width: 768px)').matches || window.getComputedStyle(sheet).position === 'absolute';
    }

    function startDrag(clientY, target = null, pointerId = null) {
        if (!isMobileSheet()) return;
        isDragging = true;
        startY = clientY;
        totalDeltaY = 0;
        startHeight = sheet.getBoundingClientRect().height;
        sheet.style.transition = 'none';

        if (pointerId !== null && target && target.setPointerCapture) {
            try {
                target.setPointerCapture(pointerId);
                capturedElement = target;
                activePointerId = pointerId;
            } catch (err) {}
        }
    }

    function moveDrag(clientY, e = null) {
        if (!isDragging) return;
        if (e && e.cancelable) {
            e.preventDefault();
        }

        const deltaY = startY - clientY;
        totalDeltaY = deltaY;
        let newHeight = startHeight + deltaY;

        const mainHeight = mainContent.getBoundingClientRect().height;
        const minHeight = Math.max(65, mainHeight * 0.12);
        const maxHeight = mainHeight * 0.95;
        
        if (newHeight < minHeight) newHeight = minHeight;
        if (newHeight > maxHeight) newHeight = maxHeight;
        
        sheet.style.height = `${newHeight}px`;
    }

    function endDrag(clientY) {
        if (!isDragging) return;
        isDragging = false;

        if (capturedElement && activePointerId !== null && capturedElement.releasePointerCapture) {
            try {
                capturedElement.releasePointerCapture(activePointerId);
            } catch (err) {}
        }
        capturedElement = null;
        activePointerId = null;

        sheet.style.transition = 'height 0.3s cubic-bezier(0.2, 0.9, 0.3, 1)'; 

        const currentHeight = sheet.getBoundingClientRect().height;
        const mainHeight = mainContent.getBoundingClientRect().height;
        const ratio = currentHeight / mainHeight;

        // 이동 거리가 8px 미만이면 '탭/클릭'으로 인식하여 높이 자동 토글
        if (Math.abs(totalDeltaY) < 8) {
            if (ratio < 0.28) {
                sheet.style.height = '45%';
            } else if (ratio < 0.70) {
                sheet.style.height = '95%';
            } else {
                sheet.style.height = '45%';
            }
            return;
        }

        // 드래그 종료 시 비율에 맞춰 부드럽게 스냅
        if (ratio > 0.65) {
            sheet.style.height = '95%';
        } else if (ratio < 0.28) {
            sheet.style.height = '15%';
        } else {
            sheet.style.height = '45%';
        }
    }

    // 드래그 트리거 영역 (핸들 바 + 주변 맛집 카운트 헤더)
    const dragTargets = [handle];
    if (listInfo) dragTargets.push(listInfo);

    // 1. 모던 포인터 이벤트 (Android Chrome, Samsung Internet, iOS Safari 등 완벽 대응)
    dragTargets.forEach(target => {
        target.addEventListener('pointerdown', (e) => {
            if (e.pointerType === 'mouse' && e.button !== 0) return;
            startDrag(e.clientY, target, e.pointerId);
        });

        target.addEventListener('pointermove', (e) => {
            moveDrag(e.clientY, e);
        });

        target.addEventListener('pointerup', (e) => {
            endDrag(e.clientY);
        });

        target.addEventListener('pointercancel', (e) => {
            endDrag(e.clientY);
        });
    });

    // 2. 터치 이벤트 fallback (특수 브라우저 및 하위 호환)
    dragTargets.forEach(target => {
        target.addEventListener('touchstart', (e) => {
            if (e.touches && e.touches.length > 0) {
                startDrag(e.touches[0].clientY);
            }
        }, { passive: true });
    });

    document.addEventListener('touchmove', (e) => {
        if (!isDragging) return;
        if (e.touches && e.touches.length > 0) {
            moveDrag(e.touches[0].clientY, e);
        }
    }, { passive: false });

    document.addEventListener('touchend', (e) => {
        if (!isDragging) return;
        const clientY = e.changedTouches && e.changedTouches.length > 0 ? e.changedTouches[0].clientY : startY;
        endDrag(clientY);
    });

    document.addEventListener('touchcancel', () => {
        if (!isDragging) return;
        endDrag(startY);
    });
}

// 초기화 및 이벤트 리스너 등록
document.addEventListener('DOMContentLoaded', () => {
    initTheme();          // 🌓 다크 모드 활성화
    initMap();
    fetchData();
    initBottomSheet();    // 📱 모바일 바텀시트 제어
    initCategoryScroll(); // 🧭 상단 카테고리 바 드래그 및 터치 스크롤 제어
    initRoulette();       // 🎲 룰렛 모달 활성화
    initSettings();       // ⚙️ 서비스 설정 모달 활성화

    // 검색 폼 제출 (Enter키 및 모바일 키보드 검색 버튼 완벽 지원)
    const searchForm = document.getElementById('search-form');
    if (searchForm) {
        searchForm.addEventListener('submit', (e) => {
            e.preventDefault();
            applySearch();
        });
    }

    const searchInput = document.getElementById('search-input');
    if (searchInput) {
        searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                applySearch();
            }
        });
    }

    // 🔍 검색 버튼 클릭 및 터치 바인딩 (모바일 터치 씹힘 방지)
    const searchBtn = document.getElementById('btn-search');
    if (searchBtn) {
        let lastSearchTrigger = 0;
        const handleSearchTrigger = (e) => {
            if (e) {
                e.preventDefault();
                e.stopPropagation();
            }
            const now = Date.now();
            if (now - lastSearchTrigger < 300) return;
            lastSearchTrigger = now;
            applySearch();
        };

        searchBtn.addEventListener('click', handleSearchTrigger);
        searchBtn.addEventListener('touchend', handleSearchTrigger);
    }

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
});