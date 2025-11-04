async function parseJsonOrThrow(response) {
  const ct = (response.headers.get('content-type') || '').toLowerCase();
  if (ct.includes('application/json')) return response.json();
  const text = await response.text();
  throw new Error(text.slice(0, 300) || 'Сервер вернул не-JSON');
}

document.addEventListener('DOMContentLoaded', () => {
  // delete
  document.querySelectorAll('.delete-zone-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-zone-id');
      const name = btn.getAttribute('data-zone-name');
      if (!confirm(`Удалить зону «${name}»?`)) return;
      try {
        const resp = await fetch(`/zones/${id}`, { method: 'DELETE' });
        const res = await parseJsonOrThrow(resp);
        if (res.success) { alert('✅ '+(res.message||'Удалено')); location.reload(); }
        else { alert('❌ '+(res.error||'Не удалось удалить')); }
      } catch (e) { alert('❌ Ошибка: ' + e.message); }
    });
  });

  // open upload modal
  document.getElementById('uploadPhotoModal')?.addEventListener('show.bs.modal', (ev) => {
    const btn = ev.relatedTarget;
    const id = btn.getAttribute('data-zone-id');
    const name = btn.getAttribute('data-zone-name');
    document.querySelector('#uploadPhotoModal .modal-title').textContent = `Загрузить фото: ${name}`;
    document.getElementById('uploadZoneId').value = id;
    document.getElementById('previewImage').classList.add('d-none');
    document.getElementById('noPreview').style.display = 'block';
    document.querySelector('#uploadPhotoForm input[name="photo"]').value = '';
  });

  // preview
  document.querySelector('#uploadPhotoForm input[name="photo"]')?.addEventListener('change', (e) => {
    const file = e.target.files[0];
    const img = document.getElementById('previewImage');
    const noPrev = document.getElementById('noPreview');
    if (file) {
      const r = new FileReader();
      r.onload = ev => { img.src = ev.target.result; img.classList.remove('d-none'); noPrev.style.display = 'none'; };
      r.readAsDataURL(file);
    } else { img.classList.add('d-none'); noPrev.style.display = 'block'; }
  });

  // upload
  document.getElementById('uploadPhotoForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const zoneId = document.getElementById('uploadZoneId').value;
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.innerHTML = '⌛ Загрузка...';
    try {
      const resp = await fetch(`/zones/${zoneId}/upload-photo`, { method: 'POST', body: new FormData(form) });
      const res = await parseJsonOrThrow(resp);
      if (res.success) {
        alert('✅ ' + (res.message || 'Фото загружено'));
        bootstrap.Modal.getInstance(document.getElementById('uploadPhotoModal')).hide();
        setTimeout(() => location.reload(), 500);
      } else { alert('❌ ' + (res.error || 'Не удалось загрузить')); }
    } catch (e2) { alert('❌ Ошибка сети: ' + e2.message); }
    finally { btn.disabled = false; btn.innerHTML = '📤 Загрузить фото'; }
  });

  // clear photo
  document.getElementById('clearPhotoBtn')?.addEventListener('click', async () => {
    const zoneId = document.getElementById('uploadZoneId').value;
    if (!zoneId) return;
    if (!confirm('Удалить фотографию зоны?')) return;
    try {
      const resp = await fetch(`/zones/${zoneId}/photo`, { method: 'DELETE' });
      const res = await parseJsonOrThrow(resp);
      if (res.success) {
        alert('🧹 Фото удалено');
        bootstrap.Modal.getInstance(document.getElementById('uploadPhotoModal')).hide();
        setTimeout(() => location.reload(), 500);
      } else { alert('❌ ' + (res.error || 'Не удалось удалить фото')); }
    } catch (e) { alert('❌ Ошибка: ' + e.message); }
  });

  // edit -> load data
  document.querySelectorAll('.edit-zone-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      const id = btn.getAttribute('data-zone-id');
      try {
        const resp = await fetch(`/api/zones/${id}`);
        const res = await parseJsonOrThrow(resp);
        if (!res.success) throw new Error(res.error || 'Не удалось получить зону');
        const z = res.zone;
        document.getElementById('editZoneId').value = z.ID || z.id || id;
        document.getElementById('editName').value = z.Name || z.name || '';
        document.getElementById('editDescription').value = z.Description || z.description || '';
        document.getElementById('editCapacity').value = z.Capacity || z.capacity || 1;
        document.getElementById('editStatus').value = z.Status || z.status || 'Доступна';
        new bootstrap.Modal(document.getElementById('editZoneModal')).show();
      } catch (e) { alert('❌ Ошибка: ' + e.message); }
    });
  });

  // edit submit
  document.getElementById('editZoneForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const id = document.getElementById('editZoneId').value;
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.innerHTML = '⌛ Сохранение...';
    const data = new URLSearchParams(new FormData(form));
    try {
      const resp = await fetch(`/zones/${id}`, { method: 'PUT', body: data });
      const res = await parseJsonOrThrow(resp);
      if (res.success) {
        alert('✅ ' + (res.message || 'Изменения сохранены'));
        bootstrap.Modal.getInstance(document.getElementById('editZoneModal')).hide();
        setTimeout(() => location.reload(), 500);
      } else { alert('❌ ' + (res.error || 'Не удалось сохранить')); }
    } catch (e2) { alert('❌ Ошибка: ' + e2.message); }
    finally { btn.disabled = false; btn.innerHTML = 'Сохранить изменения'; }
  });

  // zoom photo
  document.addEventListener('click', (e) => {
    if (!e.target.classList.contains('photo-preview')) return;
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,.8);display:flex;justify-content:center;align-items:center;z-index:9999;cursor:zoom-out;';
    const img = document.createElement('img');
    img.src = e.target.src;
    img.style.cssText = 'max-width:90%;max-height:90%;border-radius:8px;box-shadow:0 0 30px rgba(0,0,0,.5);';
    overlay.appendChild(img);
    overlay.addEventListener('click', () => document.body.removeChild(overlay));
    document.body.appendChild(overlay);
  });
});
