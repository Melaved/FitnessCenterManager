async function parseJsonOrThrow(resp) {
  const ct = (resp.headers.get('content-type') || '').toLowerCase();
  if (ct.includes('application/json')) return resp.json();
  throw new Error(await resp.text() || 'Сервер вернул не-JSON');
}
async function fill(url, select, key) {
  try {
    const resp = await fetch(url, { cache: 'no-store' });
    const data = await parseJsonOrThrow(resp);
    const arr = data[key] || [];
    select.innerHTML = '<option value="">Выберите...</option>';
    arr.forEach(x => {
      const opt = document.createElement('option');
      opt.value = x.id || x.ID;
      opt.textContent = x.name || x.Name || x.label || x.Label;
      select.appendChild(opt);
    });
  } catch (e) {
    console.error(url, e); select.innerHTML = '<option value="">Ошибка загрузки</option>';
  }
}

document.addEventListener('DOMContentLoaded', () => {
  // ===== Групповая: добавление =====
  const addGroupModal = document.getElementById('addGroupModal');
  if (addGroupModal) {
    addGroupModal.addEventListener('show.bs.modal', async () => {
      await fill('/api/trainers-for-select', document.getElementById('grpTrainer'), 'trainers');
      await fill('/api/zones-for-select',    document.getElementById('grpZone'),    'zones');
    });
    document.getElementById('addGroupForm')?.addEventListener('submit', async (e) => {
      e.preventDefault();
      const btn = e.target.querySelector('button[type="submit"]'); btn.disabled = true; btn.textContent = '⌛...';
      try {
        const resp = await fetch('/group-trainings', { method: 'POST', body: new FormData(e.target) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) { bootstrap.Modal.getInstance(addGroupModal).hide(); location.reload(); }
        else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (er) { alert('❌ ' + er.message); }
      finally { btn.disabled = false; btn.textContent = 'Сохранить'; }
    });
  }

  // ===== Групповая: редактирование =====
  document.querySelectorAll('.edit-group-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-id');
      try {
        const data = await parseJsonOrThrow(await fetch(`/api/group-trainings/${id}`, { cache:'no-store' }));
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

        new bootstrap.Modal(document.getElementById('editGroupModal')).show();
      } catch (e) { alert('❌ ' + e.message); }
    });
  });
  document.getElementById('editGroupForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id  = document.getElementById('editGroupId').value;
    const btn = e.target.querySelector('button[type="submit"]'); btn.disabled = true; btn.textContent = '⌛...';
    try {
      const body = new URLSearchParams(new FormData(e.target));
      const data = await parseJsonOrThrow(await fetch(`/group-trainings/${id}`, { method:'PUT', body }));
      if (data.success) { bootstrap.Modal.getInstance(document.getElementById('editGroupModal')).hide(); location.reload(); }
      else alert('❌ ' + (data.error || 'Ошибка'));
    } catch (e2) { alert('❌ ' + e2.message); }
    finally { btn.disabled = false; btn.textContent = 'Сохранить'; }
  });

  // ===== Групповая: удаление =====
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
      const btn = e.target.querySelector('button[type="submit"]'); btn.disabled = true; btn.textContent = '⌛...';
      try {
        const resp = await fetch('/personal-trainings', { method: 'POST', body: new FormData(e.target) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) { bootstrap.Modal.getInstance(addPersonalModal).hide(); location.reload(); }
        else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (er) { alert('❌ ' + er.message); }
      finally { btn.disabled = false; btn.textContent = 'Сохранить'; }
    });
  }

  // ===== Персональная: редактирование =====
  document.querySelectorAll('.edit-personal-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-id');
      try {
        const data = await parseJsonOrThrow(await fetch(`/api/personal-trainings/${id}`, { cache:'no-store' }));
        if (!data.success) throw new Error(data.error || 'Не найдено');
        await fill('/api/subscriptions-for-select', document.getElementById('epSub'),    'subscriptions');
        await fill('/api/trainers-for-select',      document.getElementById('epTrainer'),'trainers');

        document.getElementById('epId').value     = data.item.ID;
        document.getElementById('epSub').value    = data.item.SubscriptionID;
        document.getElementById('epTrainer').value= data.item.TrainerID;
        document.getElementById('epDate').value   = data.item.Date;
        document.getElementById('epStart').value  = data.item.StartTime;
        document.getElementById('epEnd').value    = data.item.EndTime;
        document.getElementById('epStatus').value = data.item.Status;
        document.getElementById('epPrice').value  = data.item.Price;

        new bootstrap.Modal(document.getElementById('editPersonalModal')).show();
      } catch (e) { alert('❌ ' + e.message); }
    });
  });
  document.getElementById('editPersonalForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id  = document.getElementById('epId').value;
    const btn = e.target.querySelector('button[type="submit"]'); btn.disabled = true; btn.textContent = '⌛...';
    try {
      const body = new URLSearchParams(new FormData(e.target));
      const data = await parseJsonOrThrow(await fetch(`/personal-trainings/${id}`, { method:'PUT', body }));
      if (data.success) { bootstrap.Modal.getInstance(document.getElementById('editPersonalModal')).hide(); location.reload(); }
      else alert('❌ ' + (data.error || 'Ошибка'));
    } catch (e2) { alert('❌ ' + e2.message); }
    finally { btn.disabled = false; btn.textContent = 'Сохранить'; }
  });

  // ===== Персональная: удаление =====
  document.querySelectorAll('.delete-personal-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      if (!confirm('Удалить персональную тренировку?')) return;
      const id = btn.getAttribute('data-id');
      try {
        const data = await parseJsonOrThrow(await fetch(`/personal-trainings/${id}`, { method:'DELETE' }));
        if (data.success) location.reload(); else alert('❌ ' + (data.error || 'Ошибка'));
      } catch (e) { alert('❌ ' + e.message); }
    });
  });

  // ===== Запись на групповую =====
  document.querySelectorAll('.enroll-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      document.getElementById('enGroupId').value   = btn.getAttribute('data-id');
      document.getElementById('enGroupTitle').value= btn.getAttribute('data-title') || '';
      await fill('/api/subscriptions-for-select', document.getElementById('enSub'), 'subscriptions');
      new bootstrap.Modal(document.getElementById('enrollModal')).show();
    });
  });
  document.getElementById('enrollForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = e.target.querySelector('button[type="submit"]'); btn.disabled = true; btn.textContent = '⌛...';
    try {
      const data = await parseJsonOrThrow(await fetch('/group-enrollments', { method:'POST', body: new FormData(e.target) }));
      if (data.success) { bootstrap.Modal.getInstance(document.getElementById('enrollModal')).hide(); location.reload(); }
      else alert('❌ ' + (data.error || 'Ошибка'));
    } catch (e2) { alert('❌ ' + e2.message); }
    finally { btn.disabled = false; btn.textContent = 'Создать запись'; }
  });
});
