// Wails runtime bindings will be available as window.go.main.App.*
// During development without Wails, we use mock data

const API = window.go ? window.go.main.App : null;

// --- Theme ---
function getSystemTheme() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(pref) {
  const theme = pref === 'system' ? getSystemTheme() : pref;
  document.documentElement.setAttribute('data-theme', theme);
}

const THEME_LABELS = { system: '시스템 설정', light: '라이트', dark: '다크' };

function initTheme() {
  const saved = localStorage.getItem('theme') || 'system';
  applyTheme(saved);

  const wrap = document.getElementById('theme-select-wrap');
  const trigger = document.getElementById('theme-trigger');
  const list = document.getElementById('theme-list');

  trigger.textContent = THEME_LABELS[saved] || '시스템 설정';
  updateThemeListActive(saved);

  trigger.addEventListener('click', () => {
    wrap.classList.toggle('open');
  });

  list.querySelectorAll('.custom-select-item').forEach(item => {
    item.addEventListener('click', () => {
      const val = item.dataset.value;
      trigger.textContent = THEME_LABELS[val];
      localStorage.setItem('theme', val);
      applyTheme(val);
      updateThemeListActive(val);
      wrap.classList.remove('open');
    });
  });

  document.addEventListener('click', (e) => {
    if (!wrap.contains(e.target)) wrap.classList.remove('open');
  });
}

function updateThemeListActive(val) {
  document.querySelectorAll('#theme-list .custom-select-item').forEach(item => {
    item.classList.toggle('active', item.dataset.value === val);
  });
}

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if ((localStorage.getItem('theme') || 'system') === 'system') applyTheme('system');
});

initTheme();

const DAY_NAMES = { mo: '월', tu: '화', we: '수', th: '목', fr: '금', sa: '토', su: '일' };
const DAY_KEYS = ['su', 'mo', 'tu', 'we', 'th', 'fr', 'sa'];
const DAY_LABELS = ['일', '월', '화', '수', '목', '금', '토'];
const MEAL_LABELS = { '1': '조식', '2': '중식', '3': '석식' };

let cachedDefaultTimes = null;
let cachedMealEndHours = null;
let cachedThumbRefreshWindows = null;

async function loadOperationalConfig() {
  if (!API) return;
  try {
    const [times, endHours, refreshWindows] = await Promise.all([
      API.GetDefaultTimes(),
      API.GetMealEndHours(),
      API.GetThumbRefreshWindows(),
    ]);
    if (times) cachedDefaultTimes = times;
    if (endHours) cachedMealEndHours = endHours;
    if (refreshWindows) cachedThumbRefreshWindows = refreshWindows;

    document.querySelectorAll('.meal-section[data-meal-code]').forEach(section => {
      const code = section.dataset.mealCode;
      if (cachedMealEndHours[code] != null) {
        section.dataset.endHour = cachedMealEndHours[code];
      }
    });
  } catch (e) {
    console.error('Failed to load operational config:', e);
  }
}

let state = {
  loggedIn: false,
};

let viewDate = new Date();
viewDate.setHours(0, 0, 0, 0);

function getToday() {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d;
}

function updateDateLabel() {
  const m = viewDate.getMonth() + 1;
  const d = viewDate.getDate();
  const dow = DAY_LABELS[viewDate.getDay()];
  const today = getToday();
  const isToday = viewDate.getTime() === today.getTime();
  const label = document.getElementById('date-label');
  if (isToday) {
    label.innerHTML = `${m}월 ${d}일 (${dow}) <span class="today-badge">오늘</span>`;
  } else {
    label.textContent = `${m}월 ${d}일 (${dow})`;
  }
}

function prevWeekday() {
  do { viewDate.setDate(viewDate.getDate() - 1); } while (viewDate.getDay() === 0 || viewDate.getDay() === 6);
}

function nextWeekday() {
  do { viewDate.setDate(viewDate.getDate() + 1); } while (viewDate.getDay() === 0 || viewDate.getDay() === 6);
}

document.getElementById('btn-prev-day').addEventListener('click', () => {
  prevWeekday();
  updateDateLabel();
  loadMealForDate();
});

document.getElementById('btn-next-day').addEventListener('click', () => {
  nextWeekday();
  updateDateLabel();
  loadMealForDate();
});

// --- Settings toggle ---
let settingsMode = false;

function enterSettings() {
  document.getElementById('view-meal').style.display = 'none';
  document.getElementById('view-settings').style.display = 'block';
  document.getElementById('btn-settings').style.display = 'none';
  document.getElementById('btn-close-settings').style.display = 'inline-block';
  document.getElementById('btn-reset-all').style.display = 'inline-block';
  document.querySelector('.logo').style.display = 'none';
  document.querySelector('header h1').textContent = '설정';
  settingsMode = true;
  if (API) API.SetAlwaysOnTop(true);
  checkAuthStatus();
  loadAutoStart();
  loadAppVersion();
}

function exitSettings() {
  document.getElementById('view-settings').style.display = 'none';
  document.getElementById('view-meal').style.display = '';
  document.getElementById('btn-close-settings').style.display = 'none';
  document.getElementById('btn-reset-all').style.display = 'none';
  document.getElementById('btn-settings').style.display = 'inline-block';
  document.querySelector('.logo').style.display = '';
  document.querySelector('header h1').textContent = '밥알리미';
  settingsMode = false;
  if (!loginMode && API) API.SetAlwaysOnTop(false);
}

document.getElementById('btn-settings').addEventListener('click', enterSettings);
document.getElementById('btn-close-settings').addEventListener('click', exitSettings);

// --- Auth ---
document.getElementById('login-url').addEventListener('click', (e) => {
  e.preventDefault();
  const url = e.target.href;
  if (url && url !== '#' && window.runtime) {
    window.runtime.BrowserOpenURL(url);
  }
});
document.getElementById('btn-login').addEventListener('click', startLogin);
document.getElementById('btn-logout').addEventListener('click', doLogout);
document.getElementById('btn-copy-code').addEventListener('click', () => {
  const code = document.getElementById('login-code').textContent;
  navigator.clipboard.writeText(code);
  document.getElementById('btn-copy-code').textContent = '복사됨!';
  setTimeout(() => { document.getElementById('btn-copy-code').textContent = '복사'; }, 1500);
});
document.getElementById('btn-cancel-login').addEventListener('click', () => {
  document.getElementById('login-modal').style.display = 'none';
  loginMode = false;
  if (!settingsMode && API) API.SetAlwaysOnTop(false);
});

document.getElementById('btn-close-login-modal').addEventListener('click', () => {
  document.getElementById('btn-cancel-login').click();
});

document.getElementById('btn-test-send').addEventListener('click', testSend);

const MAX_TIMES = 5;
let timeValues = [];

// --- Dynamic time chips ---
function renderTimeChips() {
  const container = document.getElementById('time-chips');
  container.innerHTML = '';
  timeValues.forEach((t, i) => {
    const chip = document.createElement('span');
    chip.className = 'time-chip';
    chip.innerHTML = `${t}<button class="chip-remove" title="제거">&times;</button>`;
    chip.querySelector('.chip-remove').addEventListener('click', () => {
      if (timeValues.length <= 1) return;
      timeValues.splice(i, 1);
      renderTimeChips();
      autoSave();
    });
    if (timeValues.length <= 1) chip.querySelector('.chip-remove').style.visibility = 'hidden';
    container.appendChild(chip);
  });
  if (timeValues.length < MAX_TIMES) {
    const addBtn = document.createElement('button');
    addBtn.className = 'time-chip-add';
    addBtn.textContent = '+';
    addBtn.addEventListener('click', () => showTimeInput(container, addBtn));
    container.appendChild(addBtn);
  }
}

function showTimeInput(container, addBtn) {
  addBtn.style.display = 'none';
  const input = document.createElement('input');
  input.type = 'time';
  input.className = 'time-chip-input';
  input.value = '12:00';
  container.appendChild(input);
  input.focus();
  const commit = () => {
    if (input.value && timeValues.length < MAX_TIMES) {
      timeValues.push(input.value);
      timeValues.sort();
    }
    renderTimeChips();
    autoSave();
  };
  input.addEventListener('blur', commit);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') input.blur();
    if (e.key === 'Escape') { input.value = ''; input.blur(); }
  });
}

function getTimeValues() {
  return timeValues.filter(v => v);
}

function setTimeRows(times) {
  timeValues = (times && times.length > 0) ? times.map(t => t.trim()) : ['08:00'];
  renderTimeChips();
}

// --- Image modal ---
function openImageModal(url) {
  const overlay = document.createElement('div');
  overlay.className = 'image-modal-overlay';
  overlay.innerHTML = `<img src="${url}">`;
  overlay.addEventListener('click', () => overlay.remove());
  document.body.appendChild(overlay);
}

// --- Meal rendering ---
function createMealCard(meal) {
  const card = document.createElement('div');
  card.className = 'meal-card';

  const hasThumb = meal.thumbnailUrl && meal.thumbnailUrl.trim() !== '';

  card.innerHTML = `
    ${hasThumb
      ? `<img class="meal-thumb" src="${meal.thumbnailUrl}" alt="${meal.name}" onerror="this.remove()">`
      : ''
    }
    <div class="meal-info">
      <div class="meal-name">${meal.name || ''}</div>
      <div class="meal-corner">${meal.corner || ''}</div>
      <div class="meal-side">${meal.side || ''}</div>
      <div class="meal-kcal">${meal.kcal ? meal.kcal + ' kcal' : ''}</div>
    </div>
  `;
  if (hasThumb) {
    card.querySelector('.meal-thumb').addEventListener('click', () => {
      openImageModal(meal.thumbnailUrl);
    });
  }
  return card;
}

// --- Meal section collapse ---
document.querySelectorAll('.meal-header').forEach(header => {
  header.addEventListener('click', () => {
    header.closest('.meal-section').classList.toggle('collapsed');
  });
});

function updateMealCollapse() {
  const isToday = viewDate.getTime() === getToday().getTime();
  document.querySelectorAll('.meal-section[data-meal-code]').forEach(section => {
    const endHour = parseInt(section.dataset.endHour || 24);
    if (isToday && new Date().getHours() >= endHour) {
      section.classList.add('collapsed');
    } else {
      section.classList.remove('collapsed');
    }
  });
}

function getWeekType(target) {
  const today = getToday();
  const todayDay = today.getDay();
  const thisMonday = new Date(today);
  thisMonday.setDate(today.getDate() - ((todayDay + 6) % 7));
  const nextMonday = new Date(thisMonday);
  nextMonday.setDate(thisMonday.getDate() + 7);
  const nextNextMonday = new Date(nextMonday);
  nextNextMonday.setDate(nextMonday.getDate() + 7);

  if (target >= thisMonday && target < nextMonday) return 1;
  if (target >= nextMonday && target < nextNextMonday) return 2;
  return 0;
}

async function loadMealForDate() {
  const sections = {
    '1': document.querySelector('#section-breakfast .meal-cards'),
    '2': document.querySelector('#section-lunch .meal-cards'),
    '3': document.querySelector('#section-dinner .meal-cards'),
  };
  const sectionIds = { '1': 'section-breakfast', '2': 'section-lunch', '3': 'section-dinner' };

  try {
    const storeCfg = API ? await API.GetStoreConfig() : null;
    if (!storeCfg || !storeCfg.idx) {
      Object.values(sections).forEach(el => { el.innerHTML = ''; });
      Object.values(sectionIds).forEach(id => { document.getElementById(id).style.display = 'none'; });
      document.getElementById('meal-empty').style.display = 'block';
      document.querySelector('#meal-empty p').textContent = '설정에서 식당을 검색해주세요.';
      return;
    }

    Object.values(sections).forEach(el => { el.innerHTML = '<div class="loading">불러오는 중</div>'; });
    const today = getToday();
    const isToday = viewDate.getTime() === today.getTime();
    let dayData = null;

    if (isToday) {
      dayData = API ? await API.GetTodayMeal() : null;
    } else {
      const weekType = getWeekType(viewDate);
      if (weekType > 0) {
        const weekData = API ? await API.GetWeekMeal(weekType) : null;
        if (weekData) {
          const dayKey = DAY_KEYS[viewDate.getDay()];
          dayData = weekData[dayKey] || null;
        }
      }
    }

    let hasAny = false;
    for (const [code, container] of Object.entries(sections)) {
      container.innerHTML = '';
      const meals = (dayData && dayData[code]) || [];
      if (meals.length > 0) {
        hasAny = true;
        meals.forEach(m => container.appendChild(createMealCard(m)));
        document.getElementById(sectionIds[code]).style.display = 'block';
      } else {
        document.getElementById(sectionIds[code]).style.display = 'none';
      }
    }

    document.getElementById('meal-empty').style.display = hasAny ? 'none' : 'block';
    updateMealCollapse();
    const mc = document.querySelector('.meal-content');
    void mc.offsetHeight;
    mc.scrollTop = 0;
  } catch (err) {
    console.error('Failed to load meal:', err);
    const msg = (err.message || String(err)).includes('설정되지') ? '설정에서 식당을 검색해주세요.' : '식단을 불러올 수 없습니다.';
    Object.values(sections).forEach(el => { el.innerHTML = ''; });
    document.getElementById('meal-empty').style.display = 'block';
    document.querySelector('#meal-empty p').textContent = msg;
  }
}

async function loadTodayMeal() {
  viewDate = getToday();
  updateDateLabel();
  await loadMealForDate();
}


// --- Direct API fetch (fallback when Wails binding not available) ---
async function fetchMealDirect(type) {
  return {};
}

async function fetchWeekDirect(weekType) {
  return {};
}

// --- Auth functions ---
async function checkAuthStatus() {
  if (!API) return;
  try {
    state.loggedIn = await API.IsLoggedIn();
    updateAuthUI();
    await loadSavedConfig();
    loadLogs();
  } catch (e) {
    console.error(e);
  }
  document.getElementById('teams-section').style.display = '';
}

async function loadSavedConfig() {
  if (!API) return;
  try {
    const cfg = await API.GetSavedConfig();
    if (!cfg) {
      setTimeRows(cachedDefaultTimes);
      return;
    }
    setTimeRows(cfg.times && cfg.times.length > 0 ? cfg.times : cachedDefaultTimes);
    if (cfg.targets && cfg.targets.length > 0) {
      selectedTargets = cfg.targets.map(t => ({
        ID: t.id, Name: t.name, Type: t.type, Group: t.group || ''
      }));
      renderTargetChips();
    }
  } catch (e) {
    console.error('Failed to load config:', e);
    setTimeRows(cachedDefaultTimes);
  }
}

function updateAuthUI() {
  const dot = document.getElementById('auth-dot');
  const stateText = document.getElementById('auth-state-text');
  const loginBtn = document.getElementById('btn-login');
  const logoutBtn = document.getElementById('btn-logout');
  const teamsConfig = document.getElementById('teams-config');

  if (state.loggedIn) {
    dot.classList.add('connected');
    stateText.textContent = '연결됨';
    stateText.style.color = 'var(--success)';
    loginBtn.style.display = 'none';
    logoutBtn.style.display = 'inline-block';
    teamsConfig.style.display = 'block';
    allTargets = [];
  } else {
    dot.classList.remove('connected');
    stateText.textContent = '연결 안 됨';
    stateText.style.color = 'var(--text-secondary)';
    loginBtn.style.display = 'inline-block';
    logoutBtn.style.display = 'none';
    teamsConfig.style.display = 'none';
  }
}

async function startLogin() {
  if (!API) {
    alert('Wails 환경에서만 로그인 가능합니다.');
    return;
  }

  loginMode = true;
  if (API) API.SetAlwaysOnTop(true);
  document.getElementById('login-modal').style.display = 'flex';
  document.getElementById('login-status').textContent = '인증 코드 생성 중...';

  try {
    const deviceCode = await API.Login();
    document.getElementById('login-url').href = deviceCode.VerificationURL;
    document.getElementById('login-url').textContent = deviceCode.VerificationURL;
    document.getElementById('login-code').textContent = deviceCode.UserCode;
    document.getElementById('login-status').textContent = '브라우저에서 코드를 입력해주세요...';

    const result = await API.WaitForLogin();
    document.getElementById('login-modal').style.display = 'none';
    state.loggedIn = true;
    updateAuthUI();
  } catch (err) {
    document.getElementById('login-status').textContent = '로그인 실패: ' + (err.message || err);
  } finally {
    loginMode = false;
    if (!settingsMode && API) API.SetAlwaysOnTop(false);
  }
}

async function doLogout() {
  if (!API) return;
  try {
    await API.Logout();
    state.loggedIn = false;
    selectedTargets = [];
    renderTargetChips();
    setTimeRows(cachedDefaultTimes);
    updateAuthUI();
  } catch (e) {
    console.error(e);
  }
}

// --- Unified target selector ---
let allTargets = [];
let selectedTargets = [];

async function loadTargets() {
  if (!API) return;
  const input = document.getElementById('target-search');
  const list = document.getElementById('target-list');
  input.value = '';
  list.innerHTML = '<div class="loading">불러오는 중</div>';
  document.getElementById('target-select').classList.add('open');

  try {
    allTargets = await API.GetAllTargets();
    renderTargetList('');
  } catch (e) {
    console.error('Failed to load targets:', e);
    list.innerHTML = '<div class="custom-select-empty">목록을 불러올 수 없습니다.</div>';
  }
}

function renderTargetList(filter) {
  const list = document.getElementById('target-list');
  list.innerHTML = '';
  const keyword = filter.toLowerCase();
  let lastGroup = null;
  const selectedIds = new Set(selectedTargets.map(s => s.ID));

  allTargets.forEach(t => {
    if (keyword && !t.Name.toLowerCase().includes(keyword) && !(t.Group && t.Group.toLowerCase().includes(keyword))) return;
    if (t.Group && t.Group !== lastGroup) {
      lastGroup = t.Group;
      const header = document.createElement('div');
      header.className = 'custom-select-group';
      header.textContent = t.Group;
      list.appendChild(header);
    } else if (!t.Group && lastGroup !== '__none__') {
      lastGroup = '__none__';
    }
    const item = document.createElement('div');
    item.className = 'custom-select-item' + (selectedIds.has(t.ID) ? ' active' : '');
    item.textContent = t.Name;
    item.addEventListener('click', () => {
      if (selectedIds.has(t.ID)) {
        selectedTargets = selectedTargets.filter(s => s.ID !== t.ID);
      } else {
        selectedTargets.push(t);
      }
      renderTargetChips();
      renderTargetList(document.getElementById('target-search').value);
      autoSave();
    });
    list.appendChild(item);
  });
  if (list.children.length === 0) {
    list.innerHTML = '<div class="custom-select-empty">결과 없음</div>';
  }
}

function renderTargetChips() {
  const container = document.getElementById('target-chips');
  container.innerHTML = '';
  selectedTargets.forEach(t => {
    const chip = document.createElement('span');
    chip.className = 'target-chip';
    const label = t.Group ? `<span class="chip-group">${t.Group} &gt;</span> ${t.Name}` : t.Name;
    chip.innerHTML = `${label}<button class="chip-remove" title="제거">&times;</button>`;
    chip.querySelector('.chip-remove').addEventListener('click', () => {
      selectedTargets = selectedTargets.filter(s => s.ID !== t.ID);
      renderTargetChips();
      renderTargetList(document.getElementById('target-search').value);
      autoSave();
    });
    container.appendChild(chip);
  });
}

const targetSearchInput = document.getElementById('target-search');
targetSearchInput.addEventListener('focus', () => {
  if (allTargets.length === 0) {
    loadTargets();
  } else {
    document.getElementById('target-select').classList.add('open');
    renderTargetList(targetSearchInput.value);
  }
});
targetSearchInput.addEventListener('input', (e) => {
  selectedTarget = null;
  renderTargetList(e.target.value);
});
document.addEventListener('click', (e) => {
  const sel = document.getElementById('target-select');
  if (!sel.contains(e.target)) sel.classList.remove('open');
});

let autoSaveTimer = null;
let saving = false;

function setSaving(v) {
  saving = v;
  const spinner = document.getElementById('save-spinner');
  const testBtn = document.getElementById('btn-test-send');
  if (v) {
    spinner.style.display = 'inline-block';
    testBtn.disabled = true;
  } else {
    spinner.style.display = 'none';
    testBtn.disabled = false;
  }
}

function autoSave() {
  if (autoSaveTimer) clearTimeout(autoSaveTimer);
  setSaving(true);
  autoSaveTimer = setTimeout(async () => {
    await saveSchedule(true);
    setSaving(false);
  }, 500);
}

let statusTimer = null;
function showStatus(msg, color, duration = 3000) {
  const el = document.getElementById('schedule-status');
  el.textContent = msg;
  el.style.color = color;
  if (statusTimer) clearTimeout(statusTimer);
  if (duration > 0) statusTimer = setTimeout(() => { el.textContent = ''; }, duration);
}

async function saveSchedule(silent) {
  if (!API) return;
  if (selectedTargets.length === 0 || getTimeValues().length === 0) {
    if (!silent) showStatus('대상과 시간을 설정해주세요.', 'var(--danger)');
    return;
  }
  const targets = selectedTargets.map(t => ({
    id: t.ID, name: t.Name, type: t.Type, group: t.Group || ''
  }));

  try {
    await API.SetSchedule(JSON.stringify(targets), getTimeValues().join(','));
    if (!silent) showStatus('저장되었습니다.', 'var(--success)');
  } catch (e) {
    if (!silent) showStatus('저장 실패: ' + (e.message || e), 'var(--danger)', 5000);
  }
}

async function testSend() {
  if (!API) return;
  if (selectedTargets.length === 0) {
    showStatus('전송 대상을 선택해주세요.', 'var(--danger)');
    return;
  }

  if (autoSaveTimer) {
    clearTimeout(autoSaveTimer);
    autoSaveTimer = null;
    await saveSchedule(true);
    setSaving(false);
  }

  showStatus('전송 중...', 'var(--text-secondary)', 0);
  let ok = 0, fail = 0, lastErr = '';
  for (const t of selectedTargets) {
    try {
      await API.SendMealToChat(t.ID);
      ok++;
    } catch (e) {
      fail++;
      lastErr = e.message || String(e);
    }
  }
  if (fail === 0) {
    showStatus(`전송 완료! (${ok}건)`, 'var(--success)');
  } else {
    showStatus(`전송 완료: 성공 ${ok}건, 실패 ${fail}건 — ${lastErr}`, 'var(--danger)', 5000);
  }
  loadLogs();
}

// --- Log viewer ---
document.getElementById('btn-open-logs').addEventListener('click', openLogWindow);

let logWindow = null;
function openLogWindow() {
  if (logWindow && !logWindow.closed) { logWindow.focus(); return; }
  logWindow = window.open('', 'logs', 'width=600,height=400');
  const doc = logWindow.document;
  const theme = document.documentElement.getAttribute('data-theme');
  const isDark = theme === 'dark';
  doc.write(`<!DOCTYPE html><html><head><title>전송 로그</title><style>
    body { font-family: 'Consolas','Courier New',monospace; font-size: 12px; margin: 0; padding: 12px;
      background: ${isDark ? '#1c1c1e' : '#faf6ef'}; color: ${isDark ? '#f0f0f0' : '#2c2c2c'}; }
    .toolbar { display:flex; gap:8px; margin-bottom:12px; }
    .toolbar button { padding:6px 12px; border:none; border-radius:6px; cursor:pointer; font-size:12px;
      background: ${isDark ? '#333' : '#e0dbd2'}; color: ${isDark ? '#f0f0f0' : '#2c2c2c'}; }
    .toolbar button:hover { opacity:0.8; }
    .entry { padding: 4px 8px; border-radius: 4px; margin-bottom: 2px; white-space: pre-wrap; word-break: break-all; }
    .entry.error { background: rgba(239,68,68,0.15); }
    .entry.warn { background: rgba(234,179,8,0.15); }
    .time { color: ${isDark ? '#6e6e75' : '#999'}; }
    .level { font-weight: bold; }
    .action { color: ${isDark ? '#ffaa44' : '#6b7c3e'}; }
    .empty { color: ${isDark ? '#6e6e75' : '#999'}; text-align: center; padding: 40px; }
  </style></head><body>
    <div class="toolbar">
      <button onclick="loadLogs()">새로고침</button>
      <button onclick="clearLogs()">초기화</button>
    </div>
    <div id="logs"></div>
  </body></html>`);
  doc.close();

  logWindow.loadLogs = async function() {
    if (!API) return;
    const container = logWindow.document.getElementById('logs');
    try {
      const logs = await API.GetLogs();
      if (!logs || logs.length === 0) { container.innerHTML = '<div class="empty">로그가 없습니다.</div>'; return; }
      container.innerHTML = '';
      for (let i = logs.length - 1; i >= 0; i--) {
        const e = logs[i];
        const div = logWindow.document.createElement('div');
        div.className = 'entry ' + e.level.toLowerCase();
        const st = e.status > 0 ? ' [' + e.status + ']' : '';
        const tgt = e.target ? ' ' + e.target.substring(0, 20) : '';
        div.innerHTML = '<span class="time">' + e.time + '</span> <span class="level">' + e.level + '</span> <span class="action">' + e.action + '</span>' + tgt + st + ' ' + e.message;
        container.appendChild(div);
      }
    } catch(e) { container.innerHTML = '<div class="empty">로그 로드 실패</div>'; }
  };

  logWindow.clearLogs = async function() {
    if (!API) return;
    await API.ClearLogs();
    logWindow.loadLogs();
  };

  logWindow.loadLogs();
}

async function loadLogs() {
  if (logWindow && !logWindow.closed) logWindow.loadLogs();
}

// --- Autostart ---
async function loadAutoStart() {
  if (!API) return;
  const chk = document.getElementById('chk-autostart');
  chk.checked = await API.GetAutoStart();
}

document.getElementById('chk-autostart').addEventListener('change', async (e) => {
  if (!API) return;
  try {
    await API.SetAutoStart(e.target.checked);
  } catch (err) {
    e.target.checked = !e.target.checked;
  }
});

// --- Confirm modal ---
function showConfirm(title, message, onOk) {
  const modal = document.getElementById('confirm-modal');
  document.getElementById('confirm-title').textContent = title;
  document.getElementById('confirm-message').textContent = message;
  modal.style.display = 'flex';
  const okBtn = document.getElementById('btn-confirm-ok');
  const cancelBtn = document.getElementById('btn-confirm-cancel');
  function cleanup() {
    modal.style.display = 'none';
    okBtn.replaceWith(okBtn.cloneNode(true));
    cancelBtn.replaceWith(cancelBtn.cloneNode(true));
  }
  okBtn.addEventListener('click', () => { cleanup(); onOk(); }, { once: true });
  cancelBtn.addEventListener('click', () => { cleanup(); }, { once: true });
}

// --- Reset all ---
document.getElementById('btn-reset-all').addEventListener('click', () => {
  if (!API) return;
  showConfirm('설정 초기화', 'Teams 인증, 전송 대상, 전송 시간, 식당 설정, 로그가 모두 삭제됩니다. 초기화하시겠습니까?', async () => {
    await API.ResetAll();
    selectedTargets = [];
    renderTargetChips();
    setTimeRows(cachedDefaultTimes);
    document.getElementById('chk-autostart').checked = false;
    document.getElementById('store-current-name').textContent = '설정 안 됨';
    document.querySelectorAll('.meal-cards').forEach(el => { el.innerHTML = ''; });
    document.querySelectorAll('.meal-section').forEach(el => { el.style.display = 'none'; });
    document.getElementById('meal-empty').style.display = 'block';
    document.querySelector('#meal-empty p').textContent = '설정에서 식당을 검색해주세요.';
    checkAuthStatus();
    showStatus('설정이 초기화되었습니다.', 'var(--success)');
  });
});

// --- ESC key to hide ---
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    if (document.getElementById('confirm-modal').style.display === 'flex') return;
    const storeModal = document.getElementById('store-modal');
    if (storeModal.style.display === 'flex') { storeModal.style.display = 'none'; return; }
    const loginModal = document.getElementById('login-modal');
    if (loginModal.style.display === 'flex') { document.getElementById('btn-cancel-login').click(); return; }
    if (settingsMode) { exitSettings(); return; }
    if (API) API.HideWindow();
  }
});

// --- Auto-hide on focus loss (debounce) ---
let hideTimer = null;
let loginMode = false;
window.addEventListener('blur', () => {
  if (loginMode || settingsMode) return;
  hideTimer = setTimeout(() => {
    if (window.go && window.go.main && window.go.main.App) {
      window.go.main.App.HideWindow();
    }
  }, 150);
});
window.addEventListener('focus', () => {
  if (hideTimer) {
    clearTimeout(hideTimer);
    hideTimer = null;
  }
});

if (window.runtime) {
  window.runtime.EventsOn('show:settings', enterSettings);
}

// --- Date change auto-refresh ---
let currentDate = new Date().toDateString();
setInterval(() => {
  const now = new Date().toDateString();
  if (now !== currentDate) {
    currentDate = now;
    viewDate = getToday();
    updateDateLabel();
    loadMealForDate();
  }
}, 60000);

// --- Thumbnail polling (lunch 11:15-11:35, dinner 17:15-17:35) ---
let thumbPollTimer = null;
function checkThumbPoll() {
  const now = new Date();
  const hm = now.getHours() * 60 + now.getMinutes();
  const inWindow = (cachedThumbRefreshWindows || []).some(w => hm >= w.start && hm <= w.end);

  if (inWindow && !thumbPollTimer) {
    thumbPollTimer = setInterval(() => {
      const today = getToday();
      if (viewDate.getTime() === today.getTime()) {
        loadMealForDate();
      }
    }, 5 * 60 * 1000);
    if (viewDate.getTime() === getToday().getTime()) loadMealForDate();
  } else if (!inWindow && thumbPollTimer) {
    clearInterval(thumbPollTimer);
    thumbPollTimer = null;
  }
}
// --- Store search ---
let storeSearchPage = 1;
let storeSearchKeyword = '';
let storeSearchTotal = 0;
const STORE_PAGE_SIZE = 10;

async function doStoreSearch(keyword, page, resultsId, pagingId) {
  if (!API || !keyword.trim()) return;
  storeSearchKeyword = keyword.trim();
  storeSearchPage = page;

  const container = document.getElementById(resultsId);
  const paging = document.getElementById(pagingId);
  container.innerHTML = '<div class="loading" style="padding:16px;">검색 중</div>';
  paging.style.display = 'none';

  try {
    const result = await API.SearchStore(storeSearchKeyword, page);
    storeSearchTotal = result.totalCount || 0;
    container.innerHTML = '';

    if (!result.storeList || result.storeList.length === 0) {
      container.innerHTML = '<div class="store-empty">검색 결과가 없습니다.</div>';
      return;
    }

    result.storeList.forEach(store => {
      const item = document.createElement('div');
      item.className = 'store-item';
      item.innerHTML = `
        <div class="store-item-name">${store.name}</div>
        <div class="store-item-info">${store.address || ''}</div>
        <div class="store-item-meta">${store.workStatus || ''} ${store.workTime ? '· ' + store.workTime : ''} ${store.hldTxt ? '· 휴무: ' + store.hldTxt : ''}</div>
      `;
      item.addEventListener('click', () => selectStore(store.idx, store.name));
      container.appendChild(item);
    });

    renderStorePaging(pagingId, resultsId);
  } catch (e) {
    container.innerHTML = '<div class="store-empty">검색 실패: ' + (e.message || e) + '</div>';
  }
}

function renderStorePaging(pagingId, resultsId) {
  const paging = document.getElementById(pagingId);
  const totalPages = Math.ceil(storeSearchTotal / STORE_PAGE_SIZE);
  if (totalPages <= 1) { paging.style.display = 'none'; return; }

  paging.style.display = 'flex';
  paging.innerHTML = '';

  const prev = document.createElement('button');
  prev.className = 'accent-btn';
  prev.textContent = '‹';
  prev.disabled = storeSearchPage <= 1;
  prev.addEventListener('click', () => doStoreSearch(storeSearchKeyword, storeSearchPage - 1, resultsId, pagingId));
  paging.appendChild(prev);

  const info = document.createElement('span');
  info.className = 'store-page-info';
  info.textContent = `${storeSearchPage} / ${totalPages}`;
  paging.appendChild(info);

  const next = document.createElement('button');
  next.className = 'accent-btn';
  next.textContent = '›';
  next.disabled = storeSearchPage >= totalPages;
  next.addEventListener('click', () => doStoreSearch(storeSearchKeyword, storeSearchPage + 1, resultsId, pagingId));
  paging.appendChild(next);
}

async function selectStore(idx, name) {
  if (!API) return;
  try {
    await API.SetStore(idx, name);
    document.getElementById('store-current-name').textContent = name;
    document.getElementById('store-modal').style.display = 'none';
    loadMealForDate();
  } catch (e) {
    console.error('Failed to set store:', e);
  }
}

async function loadStoreConfig() {
  if (!API) return;
  try {
    const cfg = await API.GetStoreConfig();
    if (cfg && cfg.idx) {
      document.getElementById('store-current-name').textContent = cfg.name || cfg.idx;
    } else {
      document.getElementById('store-current-name').textContent = '설정 안 됨';
    }
  } catch (e) {
    console.error('Failed to load store config:', e);
  }
}

function setupStoreSearch(inputId, btnId, resultsId, pagingId) {
  const input = document.getElementById(inputId);
  const btn = document.getElementById(btnId);
  const search = () => {
    if (input.value.trim()) doStoreSearch(input.value, 1, resultsId, pagingId);
  };
  btn.addEventListener('click', search);
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') search();
  });
}

setupStoreSearch('store-modal-search', 'store-modal-search-btn', 'store-modal-results', 'store-modal-paging');

document.getElementById('btn-reset-position').addEventListener('click', () => {
  if (!API) return;
  API.ResetWindowPosition();
});

document.getElementById('btn-change-store').addEventListener('click', () => {
  document.getElementById('store-modal').style.display = 'flex';
  document.getElementById('store-modal-search').value = '';
  document.getElementById('store-modal-results').innerHTML = '';
  document.getElementById('store-modal-paging').style.display = 'none';
  document.getElementById('store-modal-search').focus();
});

document.getElementById('btn-close-store-modal').addEventListener('click', () => {
  document.getElementById('store-modal').style.display = 'none';
});

// --- Update ---
let pendingUpdateInfo = null;

async function loadAppVersion() {
  if (!API) return;
  try {
    const ver = await API.GetAppVersion();
    document.getElementById('app-version').textContent = ver ? 'v' + ver : '';
  } catch (_) {}
}

async function checkForUpdate(silent) {
  if (!API) return;
  const msgEl = document.getElementById('update-message');
  const availEl = document.getElementById('update-available');
  const progressEl = document.getElementById('update-progress');

  if (!silent) msgEl.textContent = '확인 중...';
  availEl.style.display = 'none';
  progressEl.style.display = 'none';

  try {
    const info = await API.CheckForUpdate();
    pendingUpdateInfo = info;

    if (info && info.available) {
      document.getElementById('update-new-version').textContent = 'v' + info.latestVer;
      document.getElementById('update-notes').textContent = info.releaseNote || '';
      availEl.style.display = 'block';
      msgEl.textContent = '';

      if (silent) {
        const banner = document.getElementById('update-banner');
        banner.style.display = 'flex';
      }
    } else {
      if (!silent) msgEl.textContent = '최신 버전입니다.';
    }
  } catch (e) {
    if (!silent) msgEl.textContent = '확인 실패: ' + (e.message || e);
  }
}

async function performUpdate() {
  if (!pendingUpdateInfo || !pendingUpdateInfo.downloadUrl) return;
  const progressEl = document.getElementById('update-progress');
  const bar = document.getElementById('update-progress-bar');
  const text = document.getElementById('update-progress-text');
  progressEl.style.display = 'block';
  bar.style.width = '0%';
  text.textContent = '다운로드 준비 중...';

  try {
    await API.PerformUpdate(pendingUpdateInfo.downloadUrl);
  } catch (e) {
    text.textContent = '업데이트 실패: ' + (e.message || e);
  }
}

document.getElementById('btn-check-update').addEventListener('click', () => checkForUpdate(false));

document.getElementById('btn-do-update').addEventListener('click', () => {
  showConfirm('업데이트 설치', '새 버전을 설치하면 앱이 재시작됩니다. 진행하시겠습니까?', performUpdate);
});

document.getElementById('btn-release-page').addEventListener('click', () => {
  if (pendingUpdateInfo && pendingUpdateInfo.releaseUrl && API) {
    API.OpenReleasePage(pendingUpdateInfo.releaseUrl);
  }
});

document.getElementById('btn-banner-update').addEventListener('click', () => {
  document.getElementById('update-banner').style.display = 'none';
  enterSettings();
  document.getElementById('update-section').scrollIntoView({ behavior: 'smooth' });
});

document.getElementById('btn-banner-dismiss').addEventListener('click', () => {
  document.getElementById('update-banner').style.display = 'none';
});

if (window.runtime) {
  window.runtime.EventsOn('update:progress', (data) => {
    const bar = document.getElementById('update-progress-bar');
    const text = document.getElementById('update-progress-text');
    if (data && bar && text) {
      bar.style.width = data.percent + '%';
      text.textContent = data.message || '';
    }
  });

  window.runtime.EventsOn('check:update', () => {
    enterSettings();
    setTimeout(() => {
      document.getElementById('update-section').scrollIntoView({ behavior: 'smooth' });
      checkForUpdate(false);
    }, 100);
  });
}

setTimeout(() => checkForUpdate(true), 60 * 60 * 1000);
setInterval(() => checkForUpdate(true), 6 * 60 * 60 * 1000);

// --- Init ---
loadOperationalConfig().then(() => {
  setTimeRows(cachedDefaultTimes);
  updateDateLabel();
  loadStoreConfig();
  loadMealForDate();
  checkThumbPoll();
  setInterval(checkThumbPoll, 60000);
});
