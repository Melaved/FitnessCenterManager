async function parseJsonOrThrow(resp){
  const ct=(resp.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')||ct.includes('application/problem+json')) return resp.json();
  const text=await resp.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

document.addEventListener('DOMContentLoaded', ()=>{
  // CREATE
  document.getElementById('addTrainerForm')?.addEventListener('submit', async (e)=>{
    e.preventDefault();
    const form=e.currentTarget, btn=form.querySelector('button[type="submit"]');
    btn.disabled=true; btn.innerHTML='⌛ Сохранение...';
    try{
      const resp=await fetch('/trainers',{method:'POST', body:new FormData(form)});
      const res=await parseJsonOrThrow(resp);
      if(res.success){
        alert('✅ '+(res.message||'Добавлен'));
        bootstrap.Modal.getInstance(document.getElementById('addTrainerModal')).hide();
        form.reset(); location.reload();
      } else { alert('❌ '+(res.error||'Не удалось сохранить')); }
    }catch(e2){ alert('❌ '+e2.message); }
    finally{ btn.disabled=false; btn.innerHTML='Сохранить'; }
  });

  // OPEN EDIT
  document.querySelectorAll('.edit-tr-btn').forEach(btn=>{
    btn.addEventListener('click', async ()=>{
      const id=btn.getAttribute('data-tr-id');
      try{
        const resp=await fetch(`/trainers/${id}`);
        const res=await parseJsonOrThrow(resp);
        if(!res.success) throw new Error(res.error||'Не удалось получить тренера');
        const t=res.trainer;

        document.getElementById('editTrId').value = t.id;
        document.getElementById('editFio').value = t.fio||'';
        document.getElementById('editPhone').value = t.phone||'';
        document.getElementById('editSpec').value = t.specialization||'';
        document.getElementById('editHireDate').value = t.hire_date||'';
        document.getElementById('editExp').value = t.experience ?? 0;

        new bootstrap.Modal(document.getElementById('editTrainerModal')).show();
      }catch(e){ alert('❌ '+e.message); }
    });
  });

  // UPDATE
  document.getElementById('editTrainerForm')?.addEventListener('submit', async (e)=>{
    e.preventDefault();
    const form=e.currentTarget, btn=form.querySelector('button[type="submit"]');
    const id=document.getElementById('editTrId').value;
    btn.disabled=true; btn.innerHTML='⌛ Обновление...';
    try{
      const body=new URLSearchParams(new FormData(form));
      const resp=await fetch(`/trainers/${id}`,{method:'PUT', body});
      const res=await parseJsonOrThrow(resp);
      if(res.success){
        alert('✅ '+(res.message||'Обновлено'));
        bootstrap.Modal.getInstance(document.getElementById('editTrainerModal')).hide();
        location.reload();
      } else { alert('❌ '+(res.error||'Не удалось обновить')); }
    }catch(e2){ alert('❌ '+e2.message); }
    finally{ btn.disabled=false; btn.innerHTML='Обновить'; }
  });

  // DELETE
  document.querySelectorAll('.delete-tr-btn').forEach(btn=>{
    btn.addEventListener('click', async ()=>{
      const id=btn.getAttribute('data-tr-id');
      const name=btn.getAttribute('data-tr-name')||'тренер';
      if(!confirm(`Удалить «${name}»?`)) return;
      try{
        const resp=await fetch(`/trainers/${id}`,{method:'DELETE'});
        const res=await parseJsonOrThrow(resp);
        if(res.success){ alert('✅ '+(res.message||'Удалено')); location.reload(); }
        else { alert('❌ '+(res.error||'Не удалось удалить (возможно есть связанные тренировки)')); }
      }catch(e){ alert('❌ '+e.message); }
    });
  });
});
