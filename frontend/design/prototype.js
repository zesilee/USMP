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

/* ---- 接口配置表（数据驱动，编辑可回填真实值）---- */
const ifaces=[
  {name:'GE0/0/1', description:'uplink-to-core', adminStatus:'up',   mtu:9000, conv:'conv',  convLabel:'已收敛'},
  {name:'GE0/0/2', description:'',               adminStatus:'up',   mtu:1500, conv:'recon', convLabel:'收敛中'},
  {name:'GE0/0/7', description:'to-access-03',   adminStatus:'down', mtu:1500, conv:'drift', convLabel:'已漂移'},
];
function renderIfaces(){
  document.getElementById('ifBody').innerHTML=ifaces.map((r,i)=>`
    <tr>
      <td class="mono strong">${r.name}</td>
      <td>${r.description||'<span style="color:var(--ink-3)">—</span>'}</td>
      <td><span class="chip ${r.adminStatus==='up'?'conv':'off'}"><span class="glyph"></span>${r.adminStatus}</span></td>
      <td class="mono">${r.mtu}</td>
      <td><span class="chip ${r.conv}"><span class="glyph"></span>${r.convLabel}</span></td>
      <td><div class="actions"><button class="link" onclick="openDrawer(${i})">编辑</button></div></td>
    </tr>`).join('');
}
renderIfaces();

/* ---- YANG 字段模型（表单据此自动渲染）---- */
const ifmFields=[
  {key:'name',        label:'接口名', yang:'/ifm:name',         type:'text',   required:true, hint:'GE / XGE / Eth-Trunk 开头', ph:'如 GE0/0/8'},
  {key:'description', label:'描述',   yang:'/ifm:description',  type:'text',   hint:'最长 242 字符', ph:'可选'},
  {key:'adminStatus', label:'管理状态', yang:'/ifm:admin-status', type:'enum', required:true, options:['up','down']},
  {key:'mtu',         label:'MTU',    yang:'/ifm:mtu',          type:'number', required:true, hint:'46 – 9600', mono:true},
];
let formState={}, original={}, isEdit=false;
const fmt=(k,v)=> k==='description'?`"${v}"`:v;

/* ---- drawer ---- */
function openDrawer(idx){
  isEdit = typeof idx==='number';
  const seed = isEdit ? {...ifaces[idx]} : {name:'',description:'',adminStatus:'up',mtu:1500};
  formState = {...seed};
  // 编辑：以设备实际态为基线做差异；新增：基线为空 → 填入即“新增”
  original = isEdit ? {...ifaces[idx]} : {name:'',description:'',adminStatus:'',mtu:''};

  document.getElementById('drawerTitle').textContent = isEdit ? `编辑接口 · ${seed.name}` : '新增接口';
  document.getElementById('drawerSub').textContent = 'Core-Switch-01 · 10.0.0.1 · huawei-ifm';
  // 重置底部按钮（doPush 会改写）
  document.getElementById('cancelBtn').textContent='取消';
  const pb=document.getElementById('pushBtn');
  pb.style.display=''; pb.disabled=false;
  pb.innerHTML='<svg viewBox="0 0 24 24" stroke="currentColor" fill="none" stroke-width="1.7" stroke-linecap="round"><path d="M12 19V5M5 12l7-7 7 7"/></svg>下发并对账';

  renderForm();
  document.getElementById('scrim').classList.add('open');
  document.getElementById('drawer').classList.add('open');
}
function renderForm(){
  const ctrl=f=>{
    if(f.type==='enum') return `<div class="seg" data-key="${f.key}">`+
      f.options.map(o=>`<button type="button" class="seg-btn${formState[f.key]===o?' active':''}" data-v="${o}"><span class="sg"></span>${o}</button>`).join('')+`</div>`;
    return `<input class="inp${f.mono?' mono':''}" data-key="${f.key}" ${f.type==='number'?'inputmode="numeric"':''} value="${formState[f.key]??''}" placeholder="${f.ph||''}">`;
  };
  const row=f=>`<div class="form-row"><label>${f.label}${f.required?'<span class="req">*</span>':''}<span class="yp">${f.yang}</span></label>${ctrl(f)}${f.hint?`<div class="hint">${f.hint}</div>`:''}</div>`;
  const [nameF,descF,statF,mtuF]=ifmFields;
  document.getElementById('drawerBody').innerHTML=
    row(nameF)+row(descF)+
    `<div class="form-2">${row(statF)}${row(mtuF)}</div>`+
    `<button type="button" class="preview-head" id="previewHead" onclick="togglePreview()"><span>下发预览 · <b id="diffCount">0</b> 项改动</span><svg class="chev" viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"/></svg></button>`+
    `<div class="preview-body" id="previewBody"></div>`+
    `<div class="form-tip"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/></svg>字段与约束由 YANG 模型生成，校验通过才会下发，下发即触发对账。</div>`;
  // 绑定
  document.querySelectorAll('#drawerBody .inp').forEach(el=>el.addEventListener('input',()=>{
    formState[el.dataset.key]=el.value; renderPreview();
  }));
  document.querySelectorAll('#drawerBody .seg-btn').forEach(el=>el.addEventListener('click',()=>{
    const k=el.parentElement.dataset.key;
    el.parentElement.querySelectorAll('.seg-btn').forEach(b=>b.classList.remove('active'));
    el.classList.add('active'); formState[k]=el.dataset.v; renderPreview();
  }));
  renderPreview();
}
function renderPreview(){
  const changed=ifmFields.filter(f=>{
    const nv=(formState[f.key]??'').toString().trim(), ov=(original[f.key]??'').toString();
    return nv!==ov && nv!=='';
  });
  document.getElementById('diffCount').textContent=changed.length;
  document.getElementById('previewBody').innerHTML = changed.length
    ? '<div class="dva">'+changed.map(f=>{
        const nv=formState[f.key], ov=original[f.key];
        const isNew = ov===''||ov==null;
        return `<div class="dva-row"><div class="dk">${f.label}</div><div class="dv changed">`+
          (isNew?`<span class="now">${fmt(f.key,nv)}</span> <span class="tag-new">新增</span>`
                :`<span class="was">${fmt(f.key,ov)}</span><span class="arrow">→</span><span class="now">${fmt(f.key,nv)}</span>`)+
          `</div></div>`;
      }).join('')+'</div>'
    : '<div class="preview-empty">尚无改动 · 修改字段后在此预览下发差异</div>';
  // 下发按钮：有改动 + 必填齐全才可点
  const okReq = ifmFields.filter(f=>f.required).every(f=>(formState[f.key]??'').toString().trim()!=='');
  document.getElementById('pushBtn').disabled = changed.length===0 || !okReq;
}
function togglePreview(){
  document.getElementById('previewHead').classList.toggle('open');
  document.getElementById('previewBody').classList.toggle('open');
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
