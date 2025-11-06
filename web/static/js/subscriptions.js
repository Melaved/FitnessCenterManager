async function parseJsonOrThrow(response){
  const ct=(response.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')) return response.json();
  const text=await response.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

// === делегирование клика на кнопку "👥 Записанные" ===
document.addEventListener('click', async (ev) => {
  const btn = ev.target.closest('.list-enroll-btn');
  if (!btn) return;

  const groupId = btn.getAttribute('data-id');
  const title   = btn.getAttribute('data-title') || '';

  const modalEl = document.getElementById('enrollListModal');
  const titleEl = document.getElementById('enrollListTitle');
  const boxEl   = document.getElementById('enrollListContainer');

  if (!modalEl || !titleEl || !boxEl) {
    console.error('[enroll-list] Не найдены элементы модалки');
    return;
  }

  titleEl.value = `#${groupId} — ${title}`;
  boxEl.innerHTML = `<div class="text-muted">Загрузка...</div>`;

  try {
    const resp = await fetch(`/api/group-trainings/${groupId}/enrollments`, { cache:'no-store' });
    const data = await resp.json();
    if (!data.success) throw new Error(data.error || 'Ошибка загрузки');

    const list = data.enrollments || [];
    if (list.length === 0) {
      boxEl.innerHTML = `<div class="alert alert-info mb-0">Пока никто не записан.</div>`;
    } else {
      const rows = list.map((e, i) => `
        <tr>
          <td>${i+1}</td>
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
              <tr><th>#</th><th>Клиент</th><th>Абонемент</th><th>Статус</th><th>Запись</th></tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      `;
    }
  } catch (e) {
    boxEl.innerHTML = `<div class="alert alert-danger">❌ ${e.message}</div>`;
  }

  new bootstrap.Modal(modalEl).show();
});


async function fillClients(selectId, selectedId){
  try{
    const resp=await fetch('/api/clients-for-select');
    const res=await parseJsonOrThrow(resp);
    const sel=document.getElementById(selectId);
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
