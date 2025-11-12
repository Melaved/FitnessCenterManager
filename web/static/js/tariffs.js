async function parseJsonOrThrow(response){
  const ct=(response.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')||ct.includes('application/problem+json')) return response.json();
  const text=await response.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

document.addEventListener('DOMContentLoaded', () => {
  // Создание тарифа
  document.getElementById('addTariffForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.textContent = 'Сохранение...';
    try{
      const resp = await fetch('/tariffs', {method:'POST', body:new URLSearchParams(new FormData(form))});
      const res = await parseJsonOrThrow(resp);
      if(res.success){
        alert(res.message||'Тариф создан');
        bootstrap.Modal.getInstance(document.getElementById('addTariffModal'))?.hide();
        form.reset();
        location.reload();
      } else {
        alert('❌ ' + (res.error||'Ошибка сохранения'));
      }
    }catch(err){ alert('❌ ' + err.message); }
    finally{ btn.disabled=false; btn.textContent='Сохранить'; }
  });

  // Открыть модалку редактирования
  document.addEventListener('click', async (ev) => {
    const btn = ev.target.closest('.edit-tariff-btn');
    if(!btn) return;
    const id = btn.getAttribute('data-tariff-id');
    try{
      const resp = await fetch(`/api/tariffs/${id}`);
      const res = await parseJsonOrThrow(resp);
      if(!res.success) throw new Error(res.error||'Не удалось получить тариф');
      const t = res.tariff || {};
      const tid = t.id || t.ID || t['id_тарифа'] || id;
      const tname = t.name || t.Name || t['название_тарифа'] || '';
      const tdesc = t.description || t.Description || t['описание'] || '';
      const tprice = (t.price != null ? t.price : (t['стоимость'] != null ? t['стоимость'] : ''));
      const taccess = t.access_time || t.AccessTime || t['время_доступа'] || '';
      const thasGroup = !!(t.has_group_trainings || t.HasGroupTrainings || t['наличие_групповых_тренировок']);
      const thasPersonal = !!(t.has_personal_trainings || t.HasPersonalTrainings || t['наличие_персональных_тренировок']);

      document.getElementById('editTariffId').value = tid;
      document.getElementById('editName').value = tname;
      document.getElementById('editDescription').value = tdesc;
      document.getElementById('editPrice').value = tprice !== '' ? String(tprice) : '';
      document.getElementById('editAccessTime').value = taccess;
      document.getElementById('editHasGroup').checked = thasGroup;
      document.getElementById('editHasPersonal').checked = thasPersonal;
      new bootstrap.Modal(document.getElementById('editTariffModal')).show();
    }catch(err){ alert('❌ ' + err.message); }
  });

  // Сохранить изменения тарифа
  document.getElementById('editTariffForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const id = document.getElementById('editTariffId').value;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.textContent = 'Сохранение...';
    try{
      const resp = await fetch(`/tariffs/${id}`, {method:'PUT', body:new URLSearchParams(new FormData(form))});
      const res = await parseJsonOrThrow(resp);
      if(res.success){
        alert(res.message||'Сохранено');
        bootstrap.Modal.getInstance(document.getElementById('editTariffModal'))?.hide();
        location.reload();
      } else { alert('❌ ' + (res.error||'Ошибка обновления')); }
    }catch(err){ alert('❌ ' + err.message); }
    finally{ btn.disabled=false; btn.textContent='Обновить'; }
  });

  // Удаление тарифа
  document.addEventListener('click', async (ev) => {
    const btn = ev.target.closest('.delete-tariff-btn');
    if(!btn) return;
    const id = btn.getAttribute('data-tariff-id');
    const name = btn.getAttribute('data-tariff-name')||'';
    if(!confirm(`Удалить тариф «${name}»?`)) return;
    try{
      const resp = await fetch(`/tariffs/${id}`, {method:'DELETE'});
      const res = await parseJsonOrThrow(resp);
      if(res.success){ alert(res.message||'Удалено'); location.reload(); }
      else { alert('❌ ' + (res.error||'Ошибка удаления')); }
    }catch(err){ alert('❌ ' + err.message); }
  });
});
