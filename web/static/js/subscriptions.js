async function parseJsonOrThrow(response){
  const ct=(response.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')) return response.json();
  const text=await response.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

document.addEventListener('DOMContentLoaded', ()=>{
  // заполнение селектов при открытии модалки "добавить"
  document.getElementById('addSubscriptionModal')?.addEventListener('show.bs.modal', async ()=>{
    await fillClients('clientSelect');
    await fillTariffs('tariffSelect');
  });

  // создать
  document.getElementById('addSubscriptionForm')?.addEventListener('submit', async (e)=>{
    e.preventDefault();
    const form=e.currentTarget, btn=form.querySelector('button[type="submit"]');
    btn.disabled=true; btn.innerHTML='⌛ Сохранение...';
    try{
      const resp=await fetch('/subscriptions',{method:'POST', body:new FormData(form)});
      const res=await parseJsonOrThrow(resp);
      if(res.success){
        alert('✅ '+(res.message||'Создано'));
        bootstrap.Modal.getInstance(document.getElementById('addSubscriptionModal')).hide();
        form.reset();
        location.reload();
      }else{
        alert('❌ '+(res.error||'Не удалось создать'));
      }
    }catch(e2){ alert('❌ '+e2.message) }
    finally{ btn.disabled=false; btn.innerHTML='Сохранить'; }
  });

  // кн. Редактировать
  document.querySelectorAll('.edit-sub-btn').forEach(btn=>{
    btn.addEventListener('click', async ()=>{
      const id=btn.getAttribute('data-sub-id');
      try{
        // тянем одну запись
        const resp=await fetch(`/subscriptions/${id}`);
        const res=await parseJsonOrThrow(resp);
        if(!res.success) throw new Error(res.error||'Не удалось получить абонемент');
        const s=res.subscription;

        // заполняем селекты (после загрузки — выставляем выбранные)
        await fillClients('editClientSelect', s.client_id||s.ClientID);
        await fillTariffs('editTariffSelect', s.tariff_id||s.TariffID);

        // даты приводим к YYYY-MM-DD
        const toYMD = (v)=>{
          // v может быть "2025-11-04T00:00:00Z" — берём первые 10 символов
          if(typeof v==='string' && v.length>=10) return v.substring(0,10);
          const d=new Date(v); if(!isNaN(d)) return d.toISOString().substring(0,10);
          return '';
        };

        document.getElementById('editSubId').value = s.id || s.ID || id;
        document.getElementById('editStartDate').value = toYMD(s.start_date||s.StartDate);
        document.getElementById('editEndDate').value   = toYMD(s.end_date||s.EndDate);
        document.getElementById('editStatus').value    = s.status || s.Status || 'Активен';
        document.getElementById('editPrice').value     = (s.price ?? s.Price ?? '').toString();

        new bootstrap.Modal(document.getElementById('editSubscriptionModal')).show();
      }catch(e){ alert('❌ '+e.message); }
    });
  });

  // submit: обновить
  document.getElementById('editSubscriptionForm')?.addEventListener('submit', async (e)=>{
    e.preventDefault();
    const form=e.currentTarget, btn=form.querySelector('button[type="submit"]');
    const id=document.getElementById('editSubId').value;
    btn.disabled=true; btn.innerHTML='⌛ Обновление...';
    try{
      const body = new URLSearchParams(new FormData(form));
      const resp = await fetch(`/subscriptions/${id}`, { method:'PUT', body });
      const res  = await parseJsonOrThrow(resp);
      if(res.success){
        alert('✅ '+(res.message||'Обновлено'));
        bootstrap.Modal.getInstance(document.getElementById('editSubscriptionModal')).hide();
        location.reload();
      }else{
        alert('❌ '+(res.error||'Не удалось обновить'));
      }
    }catch(e2){ alert('❌ '+e2.message); }
    finally{ btn.disabled=false; btn.innerHTML='Обновить'; }
  });

  // кн. Удалить
  document.querySelectorAll('.delete-sub-btn').forEach(btn=>{
    btn.addEventListener('click', async ()=>{
      const id=btn.getAttribute('data-sub-id');
      const name=btn.getAttribute('data-client-name')||'абонемент';
      if(!confirm(`Удалить ${name}?`)) return;
      try{
        const resp=await fetch(`/subscriptions/${id}`, { method:'DELETE' });
        const res=await parseJsonOrThrow(resp);
        if(res.success){ alert('✅ '+(res.message||'Удалено')); location.reload(); }
        else{ alert('❌ '+(res.error||'Не удалось удалить')); }
      }catch(e){ alert('❌ '+e.message); }
    });
  });
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
