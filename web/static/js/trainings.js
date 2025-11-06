// /static/js/trainings.js
console.log('[trainings.js] loaded');
// ---------- утилиты ----------
async function parseJsonOrThrow(resp) {
  const ct = (resp.headers.get('content-type') || '').toLowerCase();
  if (ct.includes('application/json')) return resp.json();
  throw new Error((await resp.text()) || 'Сервер вернул не-JSON');
}
async function fill(url, select, key) {
  try {
    const resp = await fetch(url, { cache: 'no-store' });
    const data = await parseJsonOrThrow(resp);
    const arr = data[key] || [];
    select.innerHTML = '<option value="">Выберите...</option>';
    arr.forEach(x => {
      const opt = document.createElement('option');
      opt.value = x.id ?? x.ID;
      opt.textContent = x.name ?? x.Name ?? x.label ?? x.Label ?? String(opt.value);
      select.appendChild(opt);
    });
  } catch (e) {
    console.error('[fill]', url, e);
    select.innerHTML = '<option value="">Ошибка загрузки</option>';
  }
}

function buildUrlWithParams(basePath, params) {
  const u = new URL(location.origin + basePath);
  Object.entries(params).forEach(([k, v]) => {
    if (v === undefined || v === null || v === '') return;
    u.searchParams.set(k, v);
  });
  const qs = u.searchParams.toString();
  return u.pathname + (qs ? '?' + qs : '');
}

function applyFilters() {
  const params = {
    q: document.querySelector('#trSearchForm input[name="q"]')?.value?.trim(),
    trainer_id: document.getElementById('fTrainer')?.value?.trim(),
    zone_id: document.getElementById('fZone')?.value?.trim(),
    level: document.getElementById('fLevel')?.value,
    status: document.getElementById('fStatus')?.value,
    from: document.getElementById('fFrom')?.value,
    to: document.getElementById('fTo')?.value,
    upcoming: document.getElementById('fUpcoming')?.checked ? '1' : '',
    recent: document.getElementById('fRecent')?.checked ? '1' : '',
  };
  const next = buildUrlWithParams('/trainings', params);
  // надёжнее, чем заменять текущий href (меньше конфликтов с формами)
  location.assign(next);
}

// делаем доступным из HTML-атрибута onclick на кнопке
window._applyTrFilters = applyFilters;

// ---------- инициализация ----------
document.addEventListener('DOMContentLoaded', () => {
  // ===== Групповые: добавление =====
  const addGroupModal = document.getElementById('addGroupModal');
  if (addGroupModal) {
    addGroupModal.addEventListener('show.bs.modal', async () => {
      await fill('/api/trainers-for-select', document.getElementById('grpTrainer'), 'trainers');
      await fill('/api/zones-for-select',    document.getElementById('grpZone'),    'zones');
    });
    document.getElementById('addGroupForm')?.addEventListener('submit', async (e) => {
      e.preventDefault();
      const btn = e.submitter ?? e.target.querySelector('button[type="submit"]');
      if (btn) { btn.disabled = true; btn.textContent = '⌛...'; }
      try {
        const resp = await fetch('/group-trainings', { method: 'POST', body: new FormData(e.target) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) { bootstrap.Modal.getInstance(addGroupModal)?.hide(); location.reload(); }
        else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (er) { alert('❌ ' + er.message); }
      finally { if (btn) { btn.disabled = false; btn.textContent = 'Сохранить'; } }
    });
  }

// ===== Групповые: редактирование =====
document.querySelectorAll('.edit-group-btn').forEach(btn => {
  btn.addEventListener('click', async () => {
    const id = btn.getAttribute('data-id');
    try {
      const data = await parseJsonOrThrow(await fetch(`/api/group-trainings/${id}`));
      if (!data.success) throw new Error(data.error || 'Не найдено');
      
      await fill('/api/trainers-for-select', document.getElementById('egTrainer'), 'trainers');
      await fill('/api/zones-for-select',    document.getElementById('egZone'),    'zones');

      document.getElementById('editGroupId').value = data.item.ID;
      document.getElementById('egTitle').value     = data.item.Title || '';
      document.getElementById('egDesc').value      = data.item.Description || '';
      document.getElementById('egMax').value       = data.item.Max || 1;
      document.getElementById('egLevel').value     = data.item.Level || '';
      document.getElementById('egDate').value      = data.item.Date;
      document.getElementById('egStart').value     = data.item.StartTime;
      document.getElementById('egEnd').value       = data.item.EndTime;
      document.getElementById('egTrainer').value   = data.item.TrainerID;
      document.getElementById('egZone').value      = data.item.ZoneID;

      const modal = new bootstrap.Modal(document.getElementById('editGroupModal'));
      modal.show();
      
      // Добавьте обработчик для чистого закрытия
      const modalElement = document.getElementById('editGroupModal');
      modalElement.addEventListener('hidden.bs.modal', function () {
        // Очищаем форму при закрытии
        document.getElementById('editGroupForm').reset();
      });
      
    } catch (e) { 
      alert('❌ ' + e.message); 
    }
  });
});

  // ===== Групповые: удаление =====
  document.querySelectorAll('.delete-group-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      if (!confirm('Удалить групповую тренировку?')) return;
      const id = btn.getAttribute('data-id');
      try {
        const data = await parseJsonOrThrow(await fetch(`/group-trainings/${id}`, { method:'DELETE' }));
        if (data.success) location.reload(); else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (e) { alert('❌ ' + e.message); }
    });
  });

  // ===== Персональная: добавление =====
  const addPersonalModal = document.getElementById('addPersonalModal');
  if (addPersonalModal) {
    addPersonalModal.addEventListener('show.bs.modal', async () => {
      await fill('/api/subscriptions-for-select', document.getElementById('perSub'),    'subscriptions');
      await fill('/api/trainers-for-select',      document.getElementById('perTrainer'),'trainers');
    });
    document.getElementById('addPersonalForm')?.addEventListener('submit', async (e) => {
      e.preventDefault();
      const btn = e.submitter ?? e.target.querySelector('button[type="submit"]');
      if (btn) { btn.disabled = true; btn.textContent = '⌛...'; }
      try {
        const resp = await fetch('/personal-trainings', { method: 'POST', body: new FormData(e.target) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) { bootstrap.Modal.getInstance(addPersonalModal)?.hide(); location.reload(); }
        else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (er) { alert('❌ ' + er.message); }
      finally { if (btn) { btn.disabled = false; btn.textContent = 'Сохранить'; } }
    });
  }

  // ===== Персональная: редактирование/удаление =====
document.querySelectorAll('.edit-personal-btn').forEach(btn => {
  btn.addEventListener('click', async () => {
    const id = btn.getAttribute('data-id');
    try {
      const data = await parseJsonOrThrow(await fetch(`/api/personal-trainings/${id}`));
      if (!data.success) throw new Error(data.error || 'Не найдено');
      
      await fill('/api/subscriptions-for-select', document.getElementById('epSub'), 'subscriptions');
      await fill('/api/trainers-for-select',      document.getElementById('epTrainer'),'trainers');

      document.getElementById('epId').value     = data.item.ID;
      document.getElementById('epSub').value    = data.item.SubscriptionID;
      document.getElementById('epTrainer').value= data.item.TrainerID;
      document.getElementById('epDate').value   = data.item.Date;
      document.getElementById('epStart').value  = data.item.StartTime;
      document.getElementById('epEnd').value    = data.item.EndTime;
      document.getElementById('epStatus').value = data.item.Status;
      document.getElementById('epPrice').value  = data.item.Price;

      const modal = new bootstrap.Modal(document.getElementById('editPersonalModal'));
      modal.show();
      
      // Обработчик для чистого закрытия
      const modalElement = document.getElementById('editPersonalModal');
      modalElement.addEventListener('hidden.bs.modal', function () {
        document.getElementById('editPersonalForm').reset();
      });
      
    } catch (e) { 
      alert('❌ ' + e.message); 
    }
  });
});

  // ===== Запись на групповую =====
  document.querySelectorAll('.enroll-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      document.getElementById('enGroupId').value    = btn.getAttribute('data-id');
      document.getElementById('enGroupTitle').value = btn.getAttribute('data-title') || '';
      await fill('/api/subscriptions-for-select', document.getElementById('enSub'), 'subscriptions');
      new bootstrap.Modal(document.getElementById('enrollModal')).show();
    });
  });
  document.getElementById('enrollForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = e.submitter ?? e.target.querySelector('button[type="submit"]');
    if (btn) { btn.disabled = true; btn.textContent = '⌛...'; }
    try {
      const data = await parseJsonOrThrow(await fetch('/group-enrollments', { method:'POST', body: new FormData(e.target) }));
      if (data.success) { bootstrap.Modal.getInstance(document.getElementById('enrollModal'))?.hide(); location.reload(); }
      else alert('❌ ' + (data.error || 'Ошибка'));
    } catch (e2) { alert('❌ ' + e2.message); }
    finally { if (btn) { btn.disabled = false; btn.textContent = 'Создать запись'; } }
  });

  // ===== Фильтры =====
  const applyBtn = document.getElementById('applyTrFiltersBtn');
  if (applyBtn) {
    // на случай, если HTML-атрибут уберут — всё равно навесим слушатель
    applyBtn.addEventListener('click', (e) => {
      e.preventDefault();
      applyFilters();
    });
  }

  // автоприменение селекты/чекбоксы
  ['fLevel','fStatus','fUpcoming','fRecent'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('change', applyFilters);
  });

  // Enter в полях trainer/zone
  ['fTrainer','fZone'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') { e.preventDefault(); applyFilters(); }
    });
  });

  // перенос состояния фильтров в hidden при поиске
  const searchForm = document.getElementById('trSearchForm');
  if (searchForm) {
    searchForm.addEventListener('submit', () => {
      const map = {
        trainer_id: document.getElementById('fTrainer')?.value?.trim() || '',
        zone_id: document.getElementById('fZone')?.value?.trim() || '',
        level: document.getElementById('fLevel')?.value || '',
        status: document.getElementById('fStatus')?.value || '',
        from: document.getElementById('fFrom')?.value || '',
        to: document.getElementById('fTo')?.value || '',
        upcoming: document.getElementById('fUpcoming')?.checked ? '1' : '',
        recent: document.getElementById('fRecent')?.checked ? '1' : '',
      };
      Object.entries(map).forEach(([k, v]) => {
        const input = searchForm.querySelector(`input[name="${k}"]`);
        if (input) input.value = v;
      });
    });
  }
});

// === список записей на групповую ===
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('.list-enroll-btn');
  if (!btn) return;

  const groupId = btn.getAttribute('data-id');
  const title   = btn.getAttribute('data-title') || '';

  let data;
  try {
    data = await parseJsonOrThrow(
      await fetch(`/api/group-trainings/${groupId}/enrollments`, { cache: 'no-store' })
    );
    if (!data.success) throw new Error(data.error || 'Ошибка загрузки');
  } catch (err) {
    console.error(err);
    alert('❌ ' + err.message);
    return;
  }

  const modalEl = document.getElementById('enrollListModal');
  const titleEl = document.getElementById('enrollListTitle');
  const boxEl   = document.getElementById('enrollListContainer');

  if (!modalEl || !titleEl || !boxEl) {
    console.warn('Модалка списка записей не найдена в DOM');
    // как минимум покажем данные
    alert(data.enrollments?.length ? 'Записанных: ' + data.enrollments.length : 'Пока никто не записан.');
    return;
  }

  titleEl.value = `#${groupId} — ${title}`;

  const list = data.enrollments || [];
  if (list.length === 0) {
    boxEl.innerHTML = `<div class="alert alert-info mb-0">Пока никто не записан.</div>`;
  } else {
    const rows = list.map((e, idx) => `
      <tr>
        <td>${idx+1}</td>
        <td>${e.client_fio} <span class="text-muted">(#${e.client_id})</span></td>
        <td>#${e.subscription_id}</td>
        <td>
          <span class="badge ${
            e.status === 'Посетил' ? 'bg-success' :
            e.status === 'Отменил' ? 'bg-secondary' : 'bg-primary'
          }">${e.status}</span>
        </td>
        <td class="text-muted">id: ${e.id}</td>
      </tr>
    `).join('');
    boxEl.innerHTML = `
      <div class="table-responsive">
        <table class="table table-striped table-hover align-middle">
          <thead class="table-dark">
            <tr>
              <th>#</th><th>Клиент</th><th>Абонемент</th><th>Статус</th><th>Запись</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    `;
  }

  // 3) Открываем модалку 
  if (window.bootstrap?.Modal) {
    new bootstrap.Modal(modalEl).show();
  } else {
    console.warn('Bootstrap JS не найден');
  }
});


// ================== СПИСОК ЗАПИСАННЫХ (полная реализация) ==================
(() => {
  // Создаём (или берём) модалку. Если в HTML её нет — добавим динамически.
  function ensureEnrollListModal() {
    let modalEl = document.getElementById('enrollListModal');
    if (modalEl) return modalEl;

    document.body.insertAdjacentHTML(
      'beforeend',
      `
      <div class="modal fade" id="enrollListModal" tabindex="-1">
        <div class="modal-dialog modal-lg">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title">Записанные на тренировку</h5>
              <button class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <div class="mb-2">
                <input type="text" id="enrollListTitle" class="form-control" disabled>
              </div>
              <div id="enrollListContainer">
                <div class="text-muted">Загрузка...</div>
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-secondary" data-bs-dismiss="modal" type="button">Закрыть</button>
            </div>
          </div>
        </div>
      </div>
      `
    );

    modalEl = document.getElementById('enrollListModal');

    // Санитайзер — на случай, если где-то ранее создавались лишние экземпляры
    modalEl.addEventListener(
      'hidden.bs.modal',
      () => {
        // Удаляем все оставшиеся backdrop'ы и классы на body
        document.querySelectorAll('.modal-backdrop').forEach((el) => el.remove());
        document.body.classList.remove('modal-open');
        document.body.style.removeProperty('padding-right');
      },
      { once: true } // вешаем один раз
    );

    return modalEl;
  }

  // Рендер таблицы
  function renderEnrollmentsTable(boxEl, list) {
    if (!Array.isArray(list) || list.length === 0) {
      boxEl.innerHTML = `<div class="alert alert-info mb-0">Пока никто не записан.</div>`;
      return;
    }
    const rows = list
      .map(
        (e, idx) => `
        <tr>
          <td>${idx + 1}</td>
          <td>${e.client_fio} <span class="text-muted">(#${e.client_id})</span></td>
          <td>#${e.subscription_id}</td>
          <td>
            <span class="badge ${
              e.status === 'Посетил'
                ? 'bg-success'
                : e.status === 'Отменил'
                ? 'bg-secondary'
                : 'bg-primary'
            }">${e.status}</span>
          </td>
          <td class="text-muted">id: ${e.id}</td>
        </tr>
      `
      )
      .join('');

    boxEl.innerHTML = `
      <div class="table-responsive">
        <table class="table table-striped table-hover align-middle">
          <thead class="table-dark">
            <tr>
              <th>#</th><th>Клиент</th><th>Абонемент</th><th>Статус</th><th>Запись</th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    `;
  }

  // Делегированный единственный обработчик клика
  document.addEventListener('click', async (e) => {
    const btn = e.target.closest('.list-enroll-btn');
    if (!btn) return;

    const groupId = btn.getAttribute('data-id');
    const title = btn.getAttribute('data-title') || '';

    // 1) Тянем данные (Network-запрос выполнится независимо от наличия модалки в DOM)
    let data;
    try {
      const resp = await fetch(`/api/group-trainings/${groupId}/enrollments`, { cache: 'no-store' });
      const ct = (resp.headers.get('content-type') || '').toLowerCase();
      if (!ct.includes('application/json')) {
        throw new Error((await resp.text()) || 'Сервер вернул не-JSON');
      }
      data = await resp.json();
      if (!data.success) throw new Error(data.error || 'Ошибка загрузки');
    } catch (err) {
      const modalEl = ensureEnrollListModal();
      modalEl.querySelector('#enrollListTitle').value = `#${groupId} — ${title}`;
      modalEl.querySelector('#enrollListContainer').innerHTML = `
        <div class="alert alert-danger mb-0">❌ ${err.message}</div>
      `;
      if (window.bootstrap?.Modal) {
        const modal = bootstrap.Modal.getOrCreateInstance(modalEl);
        modal.show();
      }
      return;
    }

    // 2) Готовим/показываем модалку (один экземпляр!)
    const modalEl = ensureEnrollListModal();
    const titleEl = modalEl.querySelector('#enrollListTitle');
    const boxEl = modalEl.querySelector('#enrollListContainer');

    titleEl.value = `#${groupId} — ${title}`;
    renderEnrollmentsTable(boxEl, data.enrollments || []);

    if (window.bootstrap?.Modal) {
      const modal = bootstrap.Modal.getOrCreateInstance(modalEl);
      modal.show();
    } else {
      console.warn('[enroll-list] Bootstrap JS не найден — модалка не откроется.');
    }
  });
})();

