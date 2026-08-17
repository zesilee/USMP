#!/usr/bin/env node
/*
 * USMP × EviewUI 垂直切片闸门 · 离线验证工具包（方案 B）
 *
 * 用法（在离线机上）：
 *   node run.js --selftest                 # 先自检工具包本身（不碰 EviewUI）
 *   node run.js <前端工程目录> > gate-report.txt 2>&1
 *     <前端工程目录> = 含 node_modules/@nce/eview-react 的目录（如 ./frontend）
 *
 * 输出为纯文本报告（含每场景 PASS/FAIL/INFO 与 DOM 快照），整个文件带回即可。
 * 设计原则：任一场景崩溃不影响其余场景；失败时输出尽量多的现场信息，
 * 便于在线侧远程迭代下一版工具包。
 */
'use strict';
const path = require('path');
const fs = require('fs');
const Module = require('module');
const { createRequire } = Module;

const KIT = __dirname;
const kitRequire = createRequire(path.join(KIT, 'package.json'));

// ---------- 0. 运行时别名钩子：react 系 → 工具包自带 openinula ----------
const ALIAS = {
  react: 'openinula',
  'react-dom': 'openinula',
  'react-dom/client': 'openinula',
  'react-dom/test-utils': 'openinula',
  'react/jsx-runtime': 'openinula/jsx-runtime.js',
  'react/jsx-dev-runtime': 'openinula/jsx-dev-runtime.js',
  'react-intl': 'inula-intl',
  '@cloudsop/horizon': 'openinula',
  '@cloudsop/horizon-intl': 'inula-intl',
};
const aliasTarget = {};
for (const [from, to] of Object.entries(ALIAS)) {
  try {
    aliasTarget[from] = to.includes('/')
      ? path.join(path.dirname(kitRequire.resolve(to.split('/')[0] + '/package.json')), to.split('/').slice(1).join('/'))
      : kitRequire.resolve(to);
  } catch (e) {
    console.error(`[kit] 别名目标解析失败: ${from} -> ${to}: ${e.message}`);
  }
}
const origResolve = Module._resolveFilename;
Module._resolveFilename = function (request, ...rest) {
  if (aliasTarget[request]) return aliasTarget[request];
  return origResolve.call(this, request, ...rest);
};

// 样式与静态资源一律空模块（EviewUI 编译产物可能 require 样式/图片）。
for (const ext of ['.less', '.css', '.scss', '.png', '.jpg', '.gif', '.svg', '.woff', '.woff2', '.ttf', '.eot']) {
  Module._extensions[ext] = function (mod) {
    mod.exports = {};
  };
}

// ---------- 1. happy-dom 全局注册 ----------
const { Window } = kitRequire('happy-dom');
const win = new Window({ url: 'http://localhost/' });
global.window = win;
// 批量注册：window 上所有大写开头的构造器（HTML*Element/Event 族/Observer 族…）
// —— inula 事件系统会引用 HTMLInputElement 等任意全局类，缺一个就静默吞事件。
for (const k of Object.getOwnPropertyNames(win)) {
  if (!/^[A-Z]/.test(k)) continue;
  if (k in global) continue;
  try { global[k] = win[k]; } catch { /* 只读全局跳过 */ }
}
for (const k of ['document', 'location', 'history', 'customElements', 'localStorage', 'sessionStorage']) {
  try { Object.defineProperty(global, k, { value: win[k], configurable: true }); } catch {}
}
for (const k of ['getComputedStyle', 'requestAnimationFrame', 'cancelAnimationFrame']) {
  try { global[k] = win[k].bind(win); } catch {}
}
try { Object.defineProperty(global, 'navigator', { value: win.navigator, configurable: true }); } catch {}
if (!global.ResizeObserver) {
  global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
  win.ResizeObserver = global.ResizeObserver;
}
if (!win.matchMedia) {
  win.matchMedia = () => ({ matches: false, media: '', addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {}, dispatchEvent() { return false; } });
}
global.matchMedia = win.matchMedia;

// ---------- 2. 工具函数 ----------
const inula = kitRequire('openinula');
const h = inula.createElement;
const act = inula.act || ((fn) => fn());

const out = [];
function log(line) { out.push(line); console.log(line); }
function section(name) { log(`\n===== ${name} =====`); }
function result(tag, msg) { log(`### ${tag}: ${msg}`); }
const J = (x) => { try { return JSON.stringify(x); } catch { return String(x); } };

function pick(mod) { return (mod && mod.default) || mod; }

function mount(el) {
  const container = win.document.createElement('div');
  win.document.body.appendChild(container);
  const root = inula.createRoot(container);
  act(() => root.render(el));
  return {
    container,
    rerender: (next) => act(() => root.render(next)),
    unmount: () => { try { act(() => root.unmount()); } catch {} container.remove(); },
  };
}

function snap(el, n = 900) {
  const html = (el && el.innerHTML) || '';
  return html.length > n ? html.slice(0, n) + `…(共${html.length}字)` : html;
}
function bodyText() { return win.document.body.textContent || ''; }
function fire(el, type, Ctor = 'Event', init = {}) {
  const C = win[Ctor] || win.Event;
  el.dispatchEvent(new C(type, { bubbles: true, cancelable: true, ...init }));
}
function click(el) {
  act(() => {
    fire(el, 'mousedown', 'MouseEvent');
    fire(el, 'mouseup', 'MouseEvent');
    fire(el, 'click', 'MouseEvent');
  });
}
function typeInto(input, val) {
  // inula 给受控 input 的 value 装了 tracker，直接赋值会骗过变更检测导致
  // onChange 不合成——必须用原型链原生 setter（React 生态同款姿势）。
  const proto = win.HTMLTextAreaElement && input instanceof win.HTMLTextAreaElement
    ? win.HTMLTextAreaElement.prototype : win.HTMLInputElement.prototype;
  const setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
  act(() => {
    setter.call(input, val);
    fire(input, 'input', 'Event');
    fire(input, 'change', 'Event');
  });
}
// 记录回调参数：inula 合成事件在 happy-dom 下 e.target 为 null（e.currentTarget 有值），
// 防御性序列化避免回调自身抛错被吞造成"未触发"假象。
function safeArgs(args) {
  return args.map((a) => {
    if (a && typeof a === 'object') {
      if ('currentTarget' in a || 'target' in a) return '<event>';
      if (Array.isArray(a)) return JSON.stringify(a);
      return '<obj>';
    }
    return a;
  });
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
function findByText(rootEl, text) {
  const walk = (node) => {
    for (const c of node.children || []) {
      if ((c.textContent || '').trim() === text) return c;
      const hit = walk(c);
      if (hit) return hit;
    }
    return null;
  };
  return walk(rootEl);
}

// ---------- 3. selftest：不碰 EviewUI，验证工具包机制 ----------
async function selftest() {
  section('SELFTEST 工具包自检');
  log(`node: ${process.version}`);
  log(`openinula: ${kitRequire('openinula/package.json').version}`);
  function Counter() {
    const [n, setN] = inula.useState(0);
    return h('div', null,
      h('span', { 'data-test': 'count' }, `count:${n}`),
      h('button', { onClick: () => setN((v) => v + 1) }, 'inc'));
  }
  const m = mount(h(Counter));
  const btn = m.container.querySelector('button');
  click(btn);
  const txt = m.container.querySelector('[data-test="count"]').textContent;
  result('SELFTEST', txt === 'count:1' ? 'PASS（渲染+事件+状态更新 OK）' : `FAIL got=${txt}`);
  m.unmount();
}

// ---------- 4. 主流程 ----------
async function main() {
  const argv = process.argv.slice(2);
  if (argv.includes('--selftest')) { await selftest(); return; }

  const PROJ = path.resolve(argv[0] || '.');
  const nm = path.join(PROJ, 'node_modules', '@nce', 'eview-react');
  section('ENV 环境信息');
  log(`node: ${process.version}`);
  log(`工程目录: ${PROJ}`);
  log(`eview-react 存在: ${fs.existsSync(nm)}`);
  if (!fs.existsSync(nm)) {
    log('!! 找不到 node_modules/@nce/eview-react —— 请把本目录放在前端工程根旁并传入工程目录参数');
    return;
  }
  const projRequire = createRequire(path.join(PROJ, 'package.json'));
  try { log(`eview 版本: ${projRequire('@nce/eview-react/package.json').version}`); } catch (e) { log(`eview 版本读取失败: ${e.message}`); }
  try { log(`react 解析到: ${require.resolve('react')}`); } catch {}
  // locales 目录侦察（供 intl 装配与后续迭代）
  try {
    const locDir = path.join(nm, 'locales');
    log(`locales 目录: ${fs.readdirSync(locDir).slice(0, 20).join(', ')}`);
  } catch (e) { log(`locales 侦察失败: ${e.message}`); }

  await selftest();

  // intl 装配（EviewUI 组件 contextType 需要）
  const intl = kitRequire('inula-intl');
  const IntlProvider = intl.I18nProvider || intl.IntlProvider;
  let messages = {};
  for (const cand of ['zh-cn', 'zh', 'zh_CN', 'zh-cn.js', 'zh.js']) {
    try { messages = pick(projRequire(`@nce/eview-react/locales/${cand}`)); log(`intl messages 载入: locales/${cand}（${Object.keys(messages).length} 键）`); break; } catch {}
  }
  const wrap = (el) => h(IntlProvider, { locale: 'zh', messages, onError: () => '' }, el);
  const load = (sub) => pick(projRequire(`@nce/eview-react/${sub}`));

  async function scenario(name, fn) {
    section(name);
    try { await fn(); } catch (e) {
      result(name.split(' ')[0], `CRASH ${e.message}`);
      log((e.stack || '').split('\n').slice(0, 8).join('\n'));
    }
  }

  // ---- V0 冒烟：Button 挂载与点击 ----
  await scenario('V0 Button 冒烟', async () => {
    const Button = load('Button');
    let clicked = 0;
    const m = mount(wrap(h(Button, { text: 'hi', onClick: () => { clicked++; } })));
    log(`DOM: ${snap(m.container)}`);
    const btn = m.container.querySelector('button') || m.container.firstElementChild;
    click(btn);
    result('V0', clicked > 0 ? 'PASS（EviewUI 类组件在 openinula 上挂载+点击 OK）' : 'FAIL 点击未触发回调');
    m.unmount();
  });

  // ---- V1 TextField 半受控 ----
  await scenario('V1 TextField 半受控三步', async () => {
    const TextField = load('TextField');
    const changes = [];
    let setV;
    function Host() {
      const [v, s] = inula.useState('A');
      setV = s;
      return h(TextField, { value: v, onChange: (...args) => changes.push(safeArgs(args)) });
    }
    const m = mount(wrap(h(Host)));
    const input = m.container.querySelector('input');
    if (!input) { result('V1', `FAIL 未找到 input；DOM: ${snap(m.container)}`); return; }
    log(`初始 input.value=${J(input.value)}`);
    typeInto(input, 'AB');
    const afterType = input.value;
    log(`敲入后(父级拒写) input.value=${J(afterType)}  onChange args=${J(changes)}`);
    act(() => setV('Z'));
    log(`父级改 'Z' 后 input.value=${J(input.value)}（cWRP 回写有效性）`);
    act(() => setV(''));
    log(`程序化清空后 input.value=${J(input.value)}`);
    result('V1', `观察值如上——拒写后停留=${afterType === 'AB' ? '内部自改(半受控实锤)' : '已还原(受控表现)'}；onChange 参数序=${J(changes[0] || [])}`);
    m.unmount();
  });

  // ---- V2 InputSelect：弹层挂载位置 + onChange + 清空 ----
  await scenario('V2 InputSelect 弹层/受控/清空', async () => {
    const InputSelect = load('InputSelect');
    const changes = [];
    const m = mount(wrap(h(InputSelect, {
      options: [{ text: 'alpha', value: 'a' }, { text: 'beta', value: 'b' }],
      value: 'a', enableClear: true, placeholder: 'sel',
      onChange: (...args) => changes.push(safeArgs(args)),
      onClear: () => changes.push(['<onClear>']),
    })));
    log(`DOM: ${snap(m.container)}`);
    const trigger = m.container.querySelector('input') || m.container.firstElementChild;
    click(trigger);
    fire(trigger, 'focus', 'FocusEvent');
    await sleep(80);
    const inBody = findByText(win.document.body, 'beta');
    const inContainer = findByText(m.container, 'beta');
    log(`弹层选项'beta'：容器内=${!!inContainer} body级(容器外)=${!!(inBody && !inContainer)}`);
    log(`打开后 body 直接子节点数=${win.document.body.children.length}`);
    if (inBody) {
      click(inBody);
      await sleep(30);
      log(`点选 beta 后 onChange args=${J(changes)}`);
    } else {
      log(`弹层未找到——body 快照: ${snap(win.document.body, 1200)}`);
    }
    result('V2', `弹层挂载=${inContainer ? '就地(容器内)' : inBody ? 'body teleport' : '未打开(需下轮迭代触发方式)'}`);
    m.unmount();
  });

  // ---- V3 Tree：expandedKeys 同步 + onExpand ----
  await scenario('V3 Tree 受控展开', async () => {
    const Tree = load('Tree');
    const expands = [];
    const data = [{ text: 'L1', id: '1', children: [{ text: 'L2', id: '2', children: [{ text: 'L3', id: '3' }] }] }];
    let setKeys;
    function Host() {
      const [keys, s] = inula.useState([]);
      setKeys = s;
      return h(Tree, { data, expandedKeys: keys, onExpand: (...a) => expands.push(safeArgs(a)), onSelect: () => {} });
    }
    const m = mount(wrap(h(Host)));
    // 一轮报告教训：收起是 CSS 类（li.ev_tree_collapsed），textContent 恒在——改 class 口径判定。
    const liState = () => Array.from(m.container.querySelectorAll('li')).map((li) => li.className.trim()).join(' | ');
    log(`初始(expandedKeys=[]) li 类: ${liState()}`);
    act(() => setKeys(['1']));
    log(`expandedKeys=['1'] 后 li 类: ${liState()}`);
    act(() => setKeys(['1', '2']));
    log(`expandedKeys=['1','2'] 后 li 类: ${liState()}`);
    act(() => setKeys([]));
    log(`回收 [] 后 li 类: ${liState()}（受控回收有效性）`);
    // 用户点击展开箭头（一轮 DOM 快照确认元素=span.ev_tree_hit）
    const hit = m.container.querySelector('.ev_tree_hit');
    if (hit) {
      click(hit); await sleep(30);
      log(`点击 .ev_tree_hit 后 onExpand=${J(expands)}  li 类: ${liState()}`);
    } else { log('未找到 .ev_tree_hit'); }
    result('V3', 'INFO class 口径观察值如上（expanded/collapsed 类变化=受控生效）');
    m.unmount();
  });

  // ---- V4 RadioGroup：onChange 参数序 ----
  await scenario('V4 RadioGroup 参数序', async () => {
    const RadioGroup = load('RadioGroup');
    const calls = [];
    const m = mount(wrap(h(RadioGroup, {
      data: [{ value: 'a', text: 'A' }, { value: 'b', text: 'B' }],
      value: 'a', isControlled: true,
      onChange: (...args) => calls.push(safeArgs(args)),
    })));
    log(`DOM: ${snap(m.container)}`);
    // 一轮报告教训：EviewUI Radio 是 div[role=radio]+span，无原生 input——按 role 点击。
    const radios = m.container.querySelectorAll('[role="radio"]');
    log(`role=radio 数=${radios.length}`);
    const before = Array.from(radios).map((r) => r.getAttribute('aria-checked')).join(',');
    const target = radios[1];
    if (target) { click(target); await sleep(20); }
    const after = Array.from(m.container.querySelectorAll('[role="radio"]')).map((r) => r.getAttribute('aria-checked')).join(',');
    log(`aria-checked: 点前=${before} 点后=${after}`);
    log(`点 B 后 onChange calls=${J(calls)}`);
    result('V4', calls.length ? `参数序=${J(calls[0])}（判定哪个是新值）` : `未触发 onChange（aria 变化=${before !== after}）`);
    m.unmount();
  });

  // ---- V5 Loading 无 iconUrl ----
  await scenario('V5 Loading 缺省图标', async () => {
    const Loading = load('Loading');
    const m = mount(wrap(h(Loading, { isOpen: true, type: 'local', desc: 'waiting' })));
    log(`DOM: ${snap(m.container)}`);
    const img = m.container.querySelector('img');
    result('V5', `渲染成功；img=${img ? `src=${J(img.getAttribute('src'))}` : '无img元素'}`);
    m.unmount();
  });

  // ---- V6 DivMessage 自动消失 ----
  await scenario('V6 DivMessage 自动消失', async () => {
    const DivMessage = load('DivMessage');
    const m1 = mount(wrap(h(DivMessage, { text: 'msg-auto', type: 'default', disposeTimeOut: 80 })));
    log(`挂载即含文本=${bodyText().includes('msg-auto')}`);
    await sleep(400);
    const el1 = m1.container.firstElementChild;
    log(`400ms 后仍含文本=${bodyText().includes('msg-auto')} 根元素class=${el1 ? el1.className : '<无>'} style=${el1 ? el1.getAttribute('style') : ''}`);
    m1.unmount();
    const m2 = mount(wrap(h(DivMessage, { text: 'msg-keep', type: 'default', disposeTimeOut: 80, enableDisposeTimeOut: false })));
    await sleep(400);
    log(`enableDisposeTimeOut=false 400ms 后仍含文本=${bodyText().includes('msg-keep')}`);
    result('V6', 'INFO 观察值如上');
    m2.unmount();
  });

  // ---- V7 Table：动态列 render + 受控勾选 ----
  await scenario('V7 Table 动态列+勾选', async () => {
    const Table = load('Table');
    const checks = [];
    const m = mount(wrap(h(Table, {
      id: 'gate-table',
      columns: [
        { key: 'name', title: 'N', width: 100 },
        { key: 'val', title: 'V', width: 100, renderType: 'custom', render: (cv) => h('b', null, `R_${cv}`) },
      ],
      dataset: [{ name: 'r1', val: 'x1' }, { name: 'r2', val: 'x2' }],
      enableCheckBox: true, checkedRows: [],
      onRowCheck: (...args) => checks.push(safeArgs(args)),
      enablePagination: false, rowKey: 'name',
    })));
    const txt = m.container.textContent || '';
    log(`行文本 r1=${txt.includes('r1')} 自定义render R_x1=${txt.includes('R_x1')}`);
    // 一轮报告教训：EviewUI checkbox 是 div[role=checkbox]，无原生 input——按 role 点击。
    const boxes = m.container.querySelectorAll('[role="checkbox"]');
    log(`role=checkbox 数=${boxes.length}`);
    const rowBox = boxes[1] || boxes[0];
    if (rowBox) { click(rowBox); await sleep(20); }
    log(`勾第一行后 onRowCheck=${J(checks)}  aria=${Array.from(m.container.querySelectorAll('[role="checkbox"]')).map((b) => b.getAttribute('aria-checked')).join(',')}`);
    log(`DOM(截断): ${snap(m.container, 1200)}`);
    result('V7', txt.includes('R_x1') ? 'PASS 动态列 render 通' : 'FAIL 自定义 render 未出现');
    m.unmount();
  });

  section('END');
  log('报告结束——请将本输出全文带回。');
}

main().then(() => {
  // happy-dom 的内部定时器会让事件循环常驻，显式退出。
  setTimeout(() => process.exit(0), 50);
}).catch((e) => { console.error('FATAL', e); process.exit(1); });
