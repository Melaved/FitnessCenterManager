async function parseJsonOrThrow(response){
  const ct=(response.headers.get('content-type')||'').toLowerCase();
  if(ct.includes('application/json')||ct.includes('application/problem+json')) return response.json();
  const text=await response.text(); throw new Error(text.slice(0,300)||'Сервер вернул не-JSON');
}

document.addEventListener('DOMContentLoaded', function () {
  updateFilterStatus();
  initializeEditButtons();
  initializeDeleteButtons();
});

// ===== редактирование =====
function initializeEditButtons() {
  document.querySelectorAll('.edit-client-btn').forEach(button => {
    button.addEventListener('click', async function () {
      const clientId = this.getAttribute('data-client-id');
      try {
        const response = await fetch(`/clients/${clientId}`);
        const result = await parseJsonOrThrow(response);
        if (!result.success) throw new Error(result.error||'Не удалось загрузить клиента');
        const c = result.client;
        document.getElementById('editClientId').value = c.id || c.ID || clientId;
        document.getElementById('editFio').value     = c.fio || c.FIO || '';
        document.getElementById('editPhone').value   = c.phone || c.Phone || '';
        document.getElementById('editBirthDate').value = c.birth_date || c.BirthDate || '';
        document.getElementById('editMedicalData').value = c.medical_data || (c.MedicalData ? c.MedicalData.String : '') || '';
        new bootstrap.Modal(document.getElementById('editClientModal')).show();
      } catch (e) { alert('❌ '+e.message); }
    });
  });
}

document.getElementById('editClientForm')?.addEventListener('submit', async function (e) {
  e.preventDefault();
  const clientId = document.getElementById('editClientId').value;
  const btn = this.querySelector('button[type="submit"]');
  btn.disabled = true; btn.innerHTML='⌛ Обновление...';
  try {
    const data=new URLSearchParams(new FormData(this));
    const response = await fetch(`/clients/${clientId}`, { method:'PUT', body:data });
    const result = await parseJsonOrThrow(response);
    if (result.success) { alert('✅ '+(result.message||'Обновлено')); bootstrap.Modal.getInstance(document.getElementById('editClientModal')).hide(); location.reload(); }
    else { alert('❌ '+(result.error||'Не удалось обновить')); }
  } catch (e2) { alert('❌ '+e2.message); }
  finally { btn.disabled=false; btn.innerHTML='Обновить'; }
});

// ===== добавление =====
document.getElementById('addClientForm')?.addEventListener('submit', async function (e) {
  e.preventDefault();
  const btn = this.querySelector('button[type="submit"]');
  btn.disabled = true; btn.innerHTML='⌛ Сохранение...';
  try {
    const response = await fetch('/clients', { method:'POST', body:new FormData(this) });
    const result = await parseJsonOrThrow(response);
    if (result.success) { alert('✅ '+(result.message||'Сохранено')); bootstrap.Modal.getInstance(document.getElementById('addClientModal')).hide(); this.reset(); location.reload(); }
    else { alert('❌ '+(result.error||'Ошибка сохранения')); }
  } catch (e2) { alert('❌ '+e2.message); }
  finally { btn.disabled=false; btn.innerHTML='Сохранить'; }
});

// ===== удаление =====
function initializeDeleteButtons() {
  document.querySelectorAll('.delete-client-btn').forEach(button => {
    button.addEventListener('click', function () {
      const clientId = this.getAttribute('data-client-id');
      const clientName = this.getAttribute('data-client-name');
      if (confirm(`Удалить клиента "${clientName}"? Это действие нельзя отменить!`)) {
        deleteClient(clientId);
      }
    });
  });
}
async function deleteClient(clientId) {
  try{
    const response=await fetch(`/clients/${clientId}`,{method:'DELETE'});
    const result=await parseJsonOrThrow(response);
    if(result.success){ alert('✅ '+(result.message||'Удалено')); location.reload(); }
    else{ alert('❌ '+(result.error||'Не удалось удалить')); }
  }catch(e){ alert('❌ '+e.message); }
}

// ===== фильтры =====
let currentFilters = { medicalData:false, recentClients:false };
function applyFilters(){
  currentFilters.medicalData = document.getElementById('filterMedicalData').checked;
  currentFilters.recentClients = document.getElementById('filterRecentClients').checked;
  filterTable(); updateFilterStatus();
}
function clearFilters(){
  document.getElementById('filterMedicalData').checked=false;
  document.getElementById('filterRecentClients').checked=false;
  currentFilters={medicalData:false,recentClients:false};
  document.querySelectorAll('table tbody tr').forEach(row=>row.style.display='');
  updateFilterStatus(); showNoResultsMessage(false);
}
function filterTable(){
  const rows=document.querySelectorAll('table tbody tr');
  let visibleCount=0;
  rows.forEach(row=>{
    let show=true;
    if(currentFilters.medicalData){
      const badge=row.querySelector('td:nth-child(6) .badge');
      if(badge && badge.textContent.trim()==='Нет') show=false;
    }
    if(currentFilters.recentClients){
      const registerDateText=row.querySelector('td:nth-child(5)').textContent.trim();
      const [d,m,y]=registerDateText.split('.');
      const registerDate=new Date(+y,+m-1,+d);
      const limit=new Date(); limit.setDate(limit.getDate()-30);
      if(registerDate<limit) show=false;
    }
    row.style.display=show?'':'none';
    if(show) visibleCount++;
  });
  showNoResultsMessage(visibleCount===0);
}
function updateFilterStatus(){
  const el=document.getElementById('filterStatus');
  el.textContent = currentFilters.medicalData && currentFilters.recentClients ? 'С мед. данными + новые'
               : currentFilters.medicalData ? 'Только с мед. данными'
               : currentFilters.recentClients ? 'Только новые клиенты'
               : 'Все клиенты';
}
function showNoResultsMessage(show){
  let msg=document.getElementById('noResultsMessage');
  if(show && !msg){
    msg=document.createElement('div'); msg.id='noResultsMessage'; msg.className='alert alert-warning mt-3';
    msg.innerHTML=`<h5>🤷‍♂️ Клиенты не найдены</h5><p>Попробуйте изменить параметры фильтрации</p>
      <button class="btn btn-sm btn-outline-secondary" onclick="clearFilters()">Сбросить фильтры</button>`;
    const tableCard=document.querySelector('.card'); tableCard.parentNode.insertBefore(msg, tableCard.nextSibling);
  }else if(!show && msg){ msg.remove(); }
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

function applyClientFilters() {
  const params = {
    q: document.querySelector('#clientsSearchForm input[name="q"]')?.value?.trim() || '',
    medical: document.getElementById('filterMedicalData')?.checked ? '1' : '',
    recent: document.getElementById('filterRecentClients')?.checked ? '1' : '',
  };
  location.assign(buildUrlWithParams('/clients', params));
}

document.addEventListener('DOMContentLoaded', () => {
  const applyBtn = document.getElementById('applyClientsFiltersBtn');
  if (applyBtn) {
    applyBtn.addEventListener('click', (e) => {
      e.preventDefault();
      applyClientFilters();
    });
  }

  // автоматическое применение по чекбоксам
  ['filterMedicalData','filterRecentClients'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('change', applyClientFilters);
  });

  // при сабмите поисковой формы — переносим чекбоксы в hidden
  const form = document.getElementById('clientsSearchForm');
  if (form) {
    form.addEventListener('submit', () => {
      const map = {
        medical: document.getElementById('filterMedicalData')?.checked ? '1' : '',
        recent: document.getElementById('filterRecentClients')?.checked ? '1' : '',
      };
      Object.entries(map).forEach(([k, v]) => {
        const input = form.querySelector(`input[name="${k}"]`);
        if (input) input.value = v;
      });
    });
  }

});

document.addEventListener('DOMContentLoaded', function () {
  currentFilters = {
    medicalData: !!document.getElementById('filterMedicalData')?.checked,
    recentClients: !!document.getElementById('filterRecentClients')?.checked
  };
  updateFilterStatus();
  initializeEditButtons();
  initializeDeleteButtons();
});
