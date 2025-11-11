async function parseJsonOrThrow(resp) {
  const ct = (resp.headers.get('content-type') || '').toLowerCase();
  if (ct.includes('application/json')) return resp.json();
  const text = await resp.text();
  throw new Error(text || 'Сервер вернул не-JSON');
}

async function loadZonesIntoSelect(selectEl) {
  try {
    const resp = await fetch('/api/zones-for-select', { cache: 'no-store' });
    const data = await parseJsonOrThrow(resp);
    if (!data.success) throw new Error(data.error || 'Не удалось получить зоны');
    selectEl.innerHTML = '<option value="">Выберите зону...</option>';
    if (Array.isArray(data.zones) && data.zones.length) {
      for (const z of data.zones) {
        const opt = document.createElement('option');
        opt.value = z.id;
        opt.textContent = z.name;
        selectEl.appendChild(opt);
      }
    } else {
      selectEl.innerHTML = '<option value="">Зон пока нет</option>';
    }
  } catch (e) {
    console.error('❌ /api/zones-for-select:', e);
    selectEl.innerHTML = '<option value="">Ошибка загрузки зон</option>';
  }
}

function normalizeEqStatus(s) {
  s = (s || '').trim();
  if (s === 'Исправен' || s === 'Работает' || s === 'Исправно') return 'Исправен';
  if (s === 'На ремонте' || s.toLowerCase() === 'ремонт') return 'На ремонте';
  if (s === 'Списан' || s === 'Списано') return 'Списан';
  return 'Исправен';
}

document.addEventListener('DOMContentLoaded', () => {
  // tooltips
  [...document.querySelectorAll('[title]')].forEach(el => new bootstrap.Tooltip(el));

  // ===== Добавление оборудования =====
  const addModal = document.getElementById('addEquipmentModal');
  if (addModal) {
    addModal.addEventListener('show.bs.modal', () => {
      const select = document.getElementById('eqZoneSelect');
      if (select) loadZonesIntoSelect(select);
    });

    document.getElementById('addEquipmentForm')?.addEventListener('submit', async (e) => {
      e.preventDefault();
      const form = e.currentTarget;
      const btn = form.querySelector('button[type="submit"]');
      btn.disabled = true; btn.textContent = '⌛ Сохранение...';
      try {
        const resp = await fetch('/equipment', { method: 'POST', body: new FormData(form) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) {
          bootstrap.Modal.getInstance(addModal).hide();
          form.reset();
          location.reload();
        } else {
          alert('❌ ' + (data.error || 'Ошибка сохранения'));
        }
      } catch (err) {
        alert('❌ ' + err.message);
      } finally {
        btn.disabled = false; btn.textContent = 'Сохранить';
      }
    });
  }

  // ===== Загрузка/очистка фото оборудования =====
  const uploadModal = document.getElementById('uploadPhotoModal');
  if (uploadModal) {
    uploadModal.addEventListener('show.bs.modal', (ev) => {
      const btn = ev.relatedTarget;
      const id = btn.getAttribute('data-eq-id');
      const name = btn.getAttribute('data-eq-name');
      uploadModal.querySelector('.modal-title').textContent = `Загрузить фото: ${name}`;
      document.getElementById('uploadEqId').value = id;
      document.getElementById('eqPreview').classList.add('d-none');
      document.getElementById('eqNoPreview').style.display = 'block';
      uploadModal.querySelector('input[name="photo"]').value = '';
    });

    uploadModal.querySelector('input[name="photo"]').addEventListener('change', (e) => {
      const f = e.target.files[0];
      const img = document.getElementById('eqPreview');
      const noPrev = document.getElementById('eqNoPreview');
      if (f) {
        const r = new FileReader();
        r.onload = ev2 => { img.src = ev2.target.result; img.classList.remove('d-none'); noPrev.style.display = 'none'; };
        r.readAsDataURL(f);
      } else {
        img.classList.add('d-none'); noPrev.style.display = 'block';
      }
    });

    document.getElementById('uploadPhotoForm')?.addEventListener('submit', async (e) => {
      e.preventDefault();
      const id = document.getElementById('uploadEqId').value;
      const form = e.currentTarget;
      const btn = form.querySelector('button[type="submit"]');
      btn.disabled = true; btn.textContent = '⌛ Загрузка...';
      try {
        const resp = await fetch(`/equipment/${id}/upload-photo`, { method: 'POST', body: new FormData(form) });
        const data = await parseJsonOrThrow(resp);
        if (data.success) {
          bootstrap.Modal.getInstance(uploadModal).hide();
          location.reload();
        } else {
          alert('❌ ' + (data.error || 'Не удалось загрузить'));
        }
      } catch (err) {
        alert('❌ ' + err.message);
      } finally {
        btn.disabled = false; btn.textContent = '📤 Загрузить';
      }
    });

    document.getElementById('clearEqPhotoBtn')?.addEventListener('click', async () => {
      const id = document.getElementById('uploadEqId').value;
      if (!confirm('Удалить фотографию?')) return;
      try {
        const resp = await fetch(`/equipment/${id}/photo`, { method: 'DELETE' });
        const data = await parseJsonOrThrow(resp);
        if (data.success) {
          bootstrap.Modal.getInstance(uploadModal).hide();
          location.reload();
        } else {
          alert('❌ ' + (data.error || 'Не удалось удалить'));
        }
      } catch (err) {
        alert('❌ ' + err.message);
      }
    });
  }

  // ===== Редактирование оборудования =====
  document.querySelectorAll('.edit-eq-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-eq-id');
      try {
        const resp = await fetch(`/api/equipment/${id}`, { cache: 'no-store' });
        const data = await parseJsonOrThrow(resp);
        if (!data.success) throw new Error(data.error || 'Не удалось получить данные');

        document.getElementById('editEqId').value = data.item.ID;
        document.getElementById('editEqName').value = data.item.Name;
        document.getElementById('editEqPurchase').value = data.item.PurchaseDate || '';
        document.getElementById('editEqLastTO').value = data.item.LastServiceDate || '';
        document.getElementById('editEqStatus').value = normalizeEqStatus(data.item.Status);

        const select = document.getElementById('editEqZoneSelect');
        await loadZonesIntoSelect(select);
        select.value = data.item.ZoneID;

        new bootstrap.Modal(document.getElementById('editEquipmentModal')).show();
      } catch (e) {
        alert('❌ ' + e.message);
      }
    });
  });

  document.getElementById('editEquipmentForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = document.getElementById('editEqId').value;
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.textContent = '⌛ Сохранение...';
    try {
      const body = new URLSearchParams(new FormData(form));
      const resp = await fetch(`/equipment/${id}`, { method: 'PUT', body });
      const data = await parseJsonOrThrow(resp);
      if (data.success) {
        bootstrap.Modal.getInstance(document.getElementById('editEquipmentModal')).hide();
        location.reload();
      } else {
        alert('❌ ' + (data.error || 'Не удалось сохранить'));
      }
    } catch (err) {
      alert('❌ ' + err.message);
    } finally {
      btn.disabled = false; btn.textContent = 'Сохранить';
    }
  });

  // ===== Удаление оборудования =====
  document.querySelectorAll('.delete-eq-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-eq-id');
      const name = btn.getAttribute('data-eq-name');
      if (!confirm(`Удалить оборудование «${name}»?`)) return;
      try {
        const resp = await fetch(`/equipment/${id}`, { method: 'DELETE' });
        const data = await parseJsonOrThrow(resp);
        if (data.success) {
          location.reload();
        } else {
          alert('❌ ' + (data.error || 'Не удалось удалить'));
        }
      } catch (err) {
        alert('❌ ' + err.message);
      }
    });
  });

  // ===== Заявка на ремонт =====
  document.querySelectorAll('.repair-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const id = btn.getAttribute('data-eq-id');
      document.getElementById('repairEqId').value = id;
      new bootstrap.Modal(document.getElementById('repairModal')).show();
    });
  });

  document.getElementById('repairForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.textContent = '⌛ Отправка...';
    try {
      const resp = await fetch('/repairs', { method: 'POST', body: new FormData(form) });
      const data = await parseJsonOrThrow(resp);
      if (data.success) {
        bootstrap.Modal.getInstance(document.getElementById('repairModal')).hide();
        form.reset();
        location.reload();
      } else {
        alert('❌ ' + (data.error || 'Не удалось создать заявку'));
      }
    } catch (err) {
      alert('❌ ' + err.message);
    } finally {
      btn.disabled = false; btn.textContent = 'Создать заявку';
    }
  });
});

document.querySelectorAll('.repair-delete-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-repair-id');
      if (!confirm('Удалить заявку #' + id + ' ?')) return;
      try {
        const resp = await fetch(`/repairs/${id}`, { method: 'DELETE' });
        const data = await parseJsonOrThrow(resp);
        if (data.success) {
          location.reload();
        } else {
          alert('❌ ' + (data.error || 'Не удалось удалить заявку'));
        }
      } catch (err) {
        alert('❌ ' + err.message);
      }
    });
});

// показать фото заявки
document.querySelectorAll('.view-repair-photo-btn').forEach(btn => {
  btn.addEventListener('click', async () => {
    const id = btn.getAttribute('data-repair-id');
    const modalEl = document.getElementById('repairPhotoModal');
    const imgEl = document.getElementById('repairPhotoImg');
    const errEl = document.getElementById('repairPhotoError');

    imgEl.src = '';
    errEl.classList.add('d-none');

    const url = `/repairs/${id}/photo`;
    try {
      const resp = await fetch(url, { method: 'GET', cache: 'no-store' });
      if (!resp.ok) throw new Error('not ok');
      imgEl.src = url + `?t=${Date.now()}`;
    } catch {
      errEl.classList.remove('d-none');
    }

    new bootstrap.Modal(modalEl).show();
  });
});

// Загрузить фото к существующей заявке
document.querySelectorAll('.upload-repair-photo-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const id = btn.getAttribute('data-repair-id');
    document.getElementById('uploadRepairId').value = id;
    new bootstrap.Modal(document.getElementById('uploadRepairPhotoModal')).show();
  });
});

document.getElementById('uploadRepairPhotoForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.currentTarget;
  const id = document.getElementById('uploadRepairId').value;
  const btn = form.querySelector('button[type="submit"]');
  btn.disabled = true; btn.textContent = '⌛ Загрузка...';
  try {
    const resp = await fetch(`/repairs/${id}/upload-photo`, { method: 'POST', body: new FormData(form) });
    const data = await parseJsonOrThrow(resp);
    if (data.success) {
      bootstrap.Modal.getInstance(document.getElementById('uploadRepairPhotoModal')).hide();
      form.reset();
      location.reload();
    } else {
      alert('❌ ' + (data.error || 'Не удалось загрузить фото'));
    }
  } catch (err) {
    alert('❌ ' + err.message);
  } finally {
    btn.disabled = false; btn.textContent = 'Загрузить';
  }
});

// Редактировать заявку
document.querySelectorAll('.edit-repair-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const id = btn.getAttribute('data-repair-id');
    const status = btn.getAttribute('data-repair-status') || 'В работе';
    const priority = btn.getAttribute('data-repair-priority') || 'Средний';
    const desc = btn.getAttribute('data-repair-desc') || '';
    document.getElementById('editRepairId').value = id;
    document.getElementById('editRepairDesc').value = desc;
    document.getElementById('editRepairStatus').value = status;
    document.getElementById('editRepairPriority').value = priority;
    new bootstrap.Modal(document.getElementById('editRepairModal')).show();
  });
});

document.getElementById('editRepairForm')?.addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.currentTarget;
  const id = document.getElementById('editRepairId').value;
  const btn = form.querySelector('button[type="submit"]');
  btn.disabled = true; btn.textContent = '⌛ Сохранение...';
  try {
    const fd = new FormData(form);
    const resp = await fetch(`/repairs/${id}`, { method: 'PUT', body: fd });
    const data = await parseJsonOrThrow(resp);
    if (data.success) {
      bootstrap.Modal.getInstance(document.getElementById('editRepairModal')).hide();
      form.reset();
      location.reload();
    } else {
      alert('❌ ' + (data.error || 'Не удалось обновить заявку'));
    }
  } catch (err) {
    alert('❌ ' + err.message);
  } finally {
    btn.disabled = false; btn.textContent = 'Сохранить';
  }
});
