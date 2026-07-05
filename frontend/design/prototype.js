/* ---- view switching ---- */
const crumbMap={dash:'概览',devices:'设备管理',config:'配置下发',logs:'操作日志',settings:'系统设置'};
function switchView(v){
  document.querySelectorAll('.view').forEach(s=>s.classList.toggle('active',s.id==='view-'+v));
  document.querySelectorAll('.nav-item').forEach(n=>n.classList.toggle('active',n.dataset.view===v));
  document.getElementById('crumb').textContent=crumbMap[v]||'';
  document.querySelector('.scroll').scrollTop=0;
}
document.querySelectorAll('[data-view]').forEach(el=>el.addEventListener('click',()=>switchView(el.dataset.view)));

/* config sub-model switch */
const cfgTitleMap={'接口 (ifm)':'接口配置','VLAN (huawei-vlan)':'VLAN 配置','路由 (openconfig-route)':'路由配置'};
document.querySelectorAll('.nav-sub').forEach(el=>el.addEventListener('click',()=>{
  document.querySelectorAll('.nav-sub').forEach(n=>n.classList.remove('active'));
  el.classList.add('active');
  switchView('config');
  const t=cfgTitleMap[el.dataset.model]||'配置';
  document.getElementById('cfgTitle').textContent=t;
}));

/* ---- freshness ring countdown ---- */
const C=2*Math.PI*9; // circumference ~56.5
const valEl=document.getElementById('freshVal');
const secEl=document.getElementById('freshSec');
valEl.setAttribute('stroke-dasharray',C.toFixed(1));
let sec=14;
function tickFresh(){
  sec=sec>=30?0:sec+1;
  const frac=sec/30;
  valEl.setAttribute('stroke-dashoffset',(C*frac).toFixed(1));
  valEl.style.stroke = sec>24 ? 'var(--st-drift)' : 'var(--st-conv)';
  secEl.textContent=sec;
}
setInterval(tickFresh,1000);

/* ---- drawer ---- */
function openDrawer(edit){
  document.getElementById('drawerTitle').textContent=edit?'编辑接口 · GE0/0/1':'新增接口';
  document.getElementById('pushBtn').innerHTML='<svg viewBox="0 0 24 24" stroke="currentColor" fill="none" stroke-width="1.7" stroke-linecap="round"><path d="M12 19V5M5 12l7-7 7 7"/></svg>下发并对账';
  document.getElementById('scrim').classList.add('open');
  document.getElementById('drawer').classList.add('open');
}
function closeDrawer(){
  document.getElementById('scrim').classList.remove('open');
  document.getElementById('drawer').classList.remove('open');
}
document.addEventListener('keydown',e=>{if(e.key==='Escape')closeDrawer();});

/* ---- push → reconcile progress (honest, not a fake toast) ---- */
function doPush(){
  const body=document.getElementById('drawerBody');
  const btn=document.getElementById('pushBtn');
  btn.style.display='none';
  document.querySelector('.drawer-f .btn-ghost').textContent='关闭';
  body.innerHTML=`
    <div class="section-lbl" style="margin-top:0">对账进行中 · Reconciler</div>
    <div class="recon-steps">
      <div class="rstep done" id="s1"><div class="ico"><svg viewBox="0 0 24 24"><path d="M4 12l5 5L20 6"/></svg></div><div class="rstep-txt"><b>校验期望态</b><span>YANG 约束通过 · 4 字段</span></div></div>
      <div class="rline"></div>
      <div class="rstep active" id="s2"><div class="ico"></div><div class="rstep-txt"><b>编码并下发 edit-config</b><span>NETCONF SSH 830 · commit</span></div></div>
      <div class="rline"></div>
      <div class="rstep wait" id="s3"><div class="ico"></div><div class="rstep-txt"><b>回读实际态并对齐</b><span>缓存失效 · gNMI 确认</span></div></div>
    </div>`;
  setTimeout(()=>{step('s2','s3');},1400);
  setTimeout(()=>{
    document.getElementById('s3').className='rstep done';
    document.getElementById('s3').querySelector('.ico').innerHTML='<svg viewBox="0 0 24 24" stroke="#fff" fill="none" stroke-width="2.4"><path d="M4 12l5 5L20 6"/></svg>';
    body.insertAdjacentHTML('beforeend','<div style="margin-top:20px;padding:13px 15px;background:var(--st-conv-bg);border-radius:var(--r-ctl);color:var(--st-conv);font-size:13px;font-weight:600;display:flex;align-items:center;gap:9px"><span class="chip conv" style="height:auto;padding:0"><span class="glyph"></span></span>已收敛 · 期望态与实际态一致（耗时 2.8s）</div>');
  },2900);
}
function step(done,next){
  const d=document.getElementById(done);
  d.className='rstep done';
  d.querySelector('.ico').innerHTML='<svg viewBox="0 0 24 24" stroke="#fff" fill="none" stroke-width="2.4"><path d="M4 12l5 5L20 6"/></svg>';
  document.getElementById(next).className='rstep active';
}

/* ---- devices table (mock, mono facts + sparkline) ---- */
const devices=[
  ['10.0.0.1','Core-Switch-01','Huawei · CE6881','on',[3,5,4,6,5,7,6,8],'conv','已收敛','12s 前'],
  ['10.0.0.2','Core-Switch-02','Huawei · CE6881','on',[6,5,7,6,8,7,9,8],'conv','已收敛','8s 前'],
  ['10.0.2.13','Access-Switch-03','H3C · S6520','on',[2,3,2,4,3,5,4,6],'drift','已漂移','20s 前'],
  ['10.0.2.14','Access-Switch-04','H3C · S6520','on',[4,4,5,4,5,4,5,4],'conv','已收敛','5s 前'],
  ['10.0.2.17','Access-Switch-07','Huawei · S5732','on',[5,6,5,7,6,8,7,6],'recon','收敛中','刚刚'],
  ['10.0.2.21','Access-Switch-11','Cisco · C9300','off',null,'off','离线','4m 前'],
  ['10.0.2.22','Access-Switch-12','Cisco · C9300','on',[3,4,3,5,4,4,5,5],'conv','已收敛','9s 前'],
];
function spark(pts){
  if(!pts) return '<span style="color:var(--ink-3)">—</span>';
  const w=80,h=26,max=Math.max(...pts),min=Math.min(...pts);
  const nx=i=>i/(pts.length-1)*w;
  const ny=v=>h-2-(v-min)/((max-min)||1)*(h-6);
  const line=pts.map((v,i)=>`${nx(i).toFixed(1)},${ny(v).toFixed(1)}`).join(' ');
  const area=`0,${h} `+line+` ${w},${h}`;
  return `<svg class="spark" viewBox="0 0 ${w} ${h}"><polygon class="fillarea" points="${area}"/><polyline points="${line}"/></svg>`;
}
document.getElementById('devBody').innerHTML=devices.map(d=>`
  <tr>
    <td class="mono strong">${d[0]}</td>
    <td class="strong">${d[1]}</td>
    <td style="color:var(--ink-2)">${d[2]}</td>
    <td>${d[3]==='on'?'<span class="chip conv"><span class="glyph"></span>已连接</span>':'<span class="chip off"><span class="glyph"></span>断开</span>'}</td>
    <td>${spark(d[4])}</td>
    <td><span class="chip ${d[5]}"><span class="glyph"></span>${d[6]}</span></td>
    <td class="mono" style="font-size:12px;color:var(--ink-3)">${d[7]}</td>
    <td><div class="actions"><button class="link" onclick="switchView('config')">查看配置</button></div></td>
  </tr>`).join('');
