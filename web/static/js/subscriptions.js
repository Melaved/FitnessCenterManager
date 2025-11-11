async function parseJsonOrThrow(response){
  const ct=(response.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')||ct.includes('application/problem+json')) return response.json();
  const text=await response.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

async function fillClients(selectId, selectedId){
  try{
    const resp=await fetch('/api/clients-for-select');
    const res=await parseJsonOrThrow(resp);
    const sel=document.getElementById(selectId);
    if(!sel) return;
    sel.innerHTML='<option value="">Выберите клиента...</option>';
    if(res.success){
      res.clients.forEach(c=>{
        const o=document.createElement('option');
        o.value=c.id; o.textContent=c.name;
        if(selectedId && String(selectedId)===String(c.id)) o.selected=true;
        sel.appendChild(o);
      });
    }
  }catch(e){ console.error('clients-for-select', e); }
}

async function fillTariffs(selectId, selectedId){
  try{
    const resp=await fetch('/api/tariffs-for-select');
    const res=await parseJsonOrThrow(resp);
    const sel=document.getElementById(selectId);
    if(!sel) return;
    sel.innerHTML='<option value="">Выберите тариф...</option>';
    if(res.success){
      res.tariffs.forEach(t=>{
        const o=document.createElement('option');
        o.value=t.id; o.textContent=`${t.name} (${t.price} ₽)`;
        if(selectedId && String(selectedId)===String(t.id)) o.selected=true;
        sel.appendChild(o);
      });
    }
  }catch(e){ console.error('tariffs-for-select', e); }
}

document.addEventListener('DOMContentLoaded', () => {
  // селекты в модалке "Добавить"
  fillClients('clientSelect');
  fillTariffs('tariffSelect');

  // СОЗДАНИЕ
  document.getElementById('addSubscriptionForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    btn.disabled = true; btn.textContent = 'Сохранение...';
    try{
      const resp = await fetch('/subscriptions', {method:'POST', body: new URLSearchParams(new FormData(form))});
      const res = await parseJsonOrThrow(resp);
      if(res.success){
        alert(res.message||'Абонемент создан');
        bootstrap.Modal.getInstance(document.getElementById('addSubscriptionModal'))?.hide();
        form.reset();
        location.reload();
      } else {
        alert('❌ ' + (res.error||'Ошибка сохранения'));
      }
    }catch(err){ alert('❌ ' + err.message); }
    finally{ btn.disabled=false; btn.textContent='Сохранить'; }
  });

  // ОТКРЫТЬ МОДАЛКУ РЕДАКТИРОВАНИЯ (✏️)
  document.addEventListener('click', async (ev) => {
    const btn = ev.target.closest('.edit-sub-btn');
    if(!btn) return;
    const id = btn.getAttribute('data-sub-id');
    try{
      const resp = await fetch(`/subscriptions/${id}`);
      const res = await parseJsonOrThrow(resp);
      if(!res.success) throw new Error(res.error||'Не удалось получить абонемент');
      const s = res.subscription;

      await fillClients('editClientSelect', s.client_id);
      await fillTariffs('editTariffSelect', s.tariff_id);

      document.getElementById('editSubId').value = s.id;
      document.getElementById('editStartDate').value = (s.start_date||'').slice(0,10);
      document.getElementById('editEndDate').value   = (s.end_date||'').slice(0,10);
      document.getElementById('editStatus').value    = s.status || 'Активен';
      document.getElementById('editPrice').value     = s.price != null ? String(s.price) : '';

      new bootstrap.Modal(document.getElementById('editSubscriptionModal')).show();
    }catch(err){ alert('❌ ' + err.message); }
  });

  // РЕДАКТИРОВАНИЕ (submit модалки)
  document.getElementById('editSubscriptionForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.currentTarget;
    const btn = form.querySelector('button[type="submit"]');
    const id = document.getElementById('editSubId').value;
    btn.disabled = true; btn.textContent = 'Сохранение...';
    try{
      const resp = await fetch(`/subscriptions/${id}`, {method:'PUT', body: new URLSearchParams(new FormData(form))});
      const res = await parseJsonOrThrow(resp);
      if(res.success){
        alert(res.message||'Сохранено');
        bootstrap.Modal.getInstance(document.getElementById('editSubscriptionModal'))?.hide();
        location.reload();
      } else {
        alert('❌ ' + (res.error||'Ошибка обновления'));
      }
    }catch(err){ alert('❌ ' + err.message); }
    finally{ btn.disabled=false; btn.textContent='Обновить'; }
  });

  // УДАЛЕНИЕ (🗑️)
  document.addEventListener('click', async (ev) => {
    const btn = ev.target.closest('.delete-sub-btn');
    if(!btn) return;
    const id = btn.getAttribute('data-sub-id');
    const clientName = btn.getAttribute('data-client-name')||'';
    if(!confirm(`Удалить абонемент #${id} клиента «${clientName}»?`)) return;
    try{
      const resp = await fetch(`/subscriptions/${id}`, {method:'DELETE'});
      const res = await parseJsonOrThrow(resp);
      if(res.success){ alert(res.message||'Удалено'); location.reload(); }
      else { alert('❌ ' + (res.error||'Ошибка удаления')); }
    }catch(err){ alert('❌ ' + err.message); }
  });
});
