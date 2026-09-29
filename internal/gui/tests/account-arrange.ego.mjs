// Start account-arrange-fixture.cjs, then:
// ego-browser nodejs < internal/gui/tests/account-arrange.ego.mjs
// For another round, replace the taskSpace argument with the printed space ID.
const assert = (await import('node:assert/strict')).default;
const task = await taskSpace('Magpie account arrangement regression');
const page = task.page('p1');
console.log({spaceId:task.spaceId});
const ids = () => page.evaluate(()=>[...document.querySelectorAll('.accts [data-account-id]')].map(r=>r.dataset.accountId));
const settled = () => page.waitForFunction(()=>!accountArranging && !accountSaving);
const audit = async () => JSON.parse((await page.fetch('/__qa')).body);
const open = async id => {
 await page.goto(`http://127.0.0.1:18427/?view=providers&edit=${id}`);
 await page.waitForSelector('.accts [data-account-id]');
 await page.evaluate(()=>{window.qaErrors=[];addEventListener('error',e=>qaErrors.push(e.message));addEventListener('unhandledrejection',e=>qaErrors.push(String(e.reason)));});
};
const topRows = async()=>{
 await page.hover('.accts [data-account-id]:first-child');
 return page.evaluate(()=>[...document.querySelectorAll('.accts [data-account-id]')].slice(0,3).map(r=>r.getBoundingClientRect().toJSON()));
};
const drag = async(cancel=false)=>{
 const [a,b]=await topRows();
 await page.mouse.move(a.left+150,a.top+a.height/2);
 await page.mouse.down();
 await page.mouse.move(b.left+150,b.bottom-2,{steps:12});
 await page.waitForSelector('.acc.dragging');
 if(cancel) await page.keyboard.press('Escape');
 await page.mouse.up(); await settled();
};
for(const id of ['antigravity','relay']) {
 await open(id);
 for(const [mode,label] of [['','智能'],['order','按顺序'],['rotate','轮流'],['usage','用量少的优先']]) {
  await page.click(`.editor .segs button:text-is("${label}")`);
  await page.waitForFunction(({id,mode})=>providers.providers.find(p=>p.id===id).routing===mode,{id,mode});
  const before=await ids(), count=(await audit()).posted.filter(p=>p.action.endsWith('/arrange')).length;
  await drag();
  const want=[before[1],before[0],...before.slice(2)];
  assert.deepEqual(await ids(),want,`${id} ${mode}: pointer reorder`);
  const data=await audit();
  assert.equal(data.posted.filter(p=>p.action.endsWith('/arrange')).length,count+1,'one write per drop');
  assert.equal(data.providers.providers.find(p=>p.id===id).routing,mode,'routing policy unchanged');
  await page.reload();await page.waitForSelector('.accts [data-account-id]');
  assert.deepEqual(await ids(),want,'order survives reload');
 }
 const before=await ids(), count=(await audit()).posted.length;
 await drag(true);
 assert.deepEqual(await ids(),before,'Escape cancels');
 assert.equal((await audit()).posted.length,count,'cancel does not save');
 await page.focus('.accts [data-account-id]:first-child');
 await page.keyboard.press('Alt+ArrowDown');await settled();
 assert.deepEqual(await ids(),[before[1],before[0],...before.slice(2)],'keyboard reorder');
 assert.equal(await page.evaluate(()=>document.activeElement.dataset.accountId),before[0],'keyboard focus retained');
 const saved=await ids();await page.fetch('/__fail');await drag();
 assert.deepEqual(await ids(),saved,'failed save rolls back');
 assert.equal(await page.evaluate(()=>document.querySelectorAll('.accts .grip, .accts .ag-handle').length),0,'no handles');
 console.log(`${id}: four routing modes, reload, cancel, keyboard, rollback PASS`);
}
// Key names remain clickable, but dragging one must not open its rename field.
await open('relay');
await page.click('.accts [data-account-id]:first-child .rename');
assert.equal(await page.evaluate(()=>!!document.querySelector('.rename-in')),true,'rename click works');
await page.keyboard.press('Escape');
const before=await ids();
const names=await page.evaluate(()=>[...document.querySelectorAll('.accts .rename')].slice(0,2).map(r=>r.getBoundingClientRect().toJSON()));
await page.mouse.move(names[0].left+names[0].width/2,names[0].top+names[0].height/2);
await page.mouse.down();await page.mouse.move(names[1].left+names[1].width/2,names[1].bottom+8,{steps:12});
await page.waitForSelector('.acc.dragging');
await page.mouse.up();await settled();
assert.equal(await page.evaluate(()=>!!document.querySelector('.rename-in')),false,'drag does not rename');
assert.notDeepEqual(await ids(),before);
// Single rows have neither drag affordance nor a sorting tab stop.
await page.evaluate(()=>{const p=providers.providers.find(p=>p.id==='relay');p.keyList=p.keyList.slice(0,1);renderProviders();});
assert.equal(await page.evaluate(()=>document.querySelector('.accts').classList.contains('reorderable')),false);
assert.equal(await page.evaluate(()=>document.querySelector('.accts [data-account-id]').tabIndex),-1);
await open('antigravity');
// A background quota refresh cannot replace the captured row; drag near the
// scroll viewport's edge to reach accounts initially below the fold.
const [a]=await topRows();
const bounds=await page.evaluate(()=>document.querySelector('.ebody').getBoundingClientRect().toJSON());
await page.mouse.move(a.left+150,a.top+a.height/2);await page.mouse.down();
await page.mouse.move(a.left+150,bounds.bottom-4,{steps:30});
await page.waitForSelector('.acc.dragging');
await page.waitForFunction(()=>document.querySelector('.ebody').scrollTop>100, null, {timeout:5000});
const refresh=await page.evaluate(()=>{const held=document.querySelector('.acc.dragging');renderProviders();return {same:held===document.querySelector('.acc.dragging'),deferred:accountRenderPending};});
assert(refresh.same && refresh.deferred,'background redraw deferred');
await page.mouse.up();await settled();
assert.deepEqual(await page.evaluate(()=>window.qaErrors),[]);
console.log('rename, single account, autoscroll, background refresh PASS');
// The shared primitive must preserve the existing agent arrangement too.
await page.goto('http://127.0.0.1:18427/?view=agents');
await page.waitForSelector('#agents > .agent');
const agents=await page.evaluate(()=>[...document.querySelectorAll('#agents > .agent')].map(r=>({id:r.dataset.id,handle:r.querySelector('.ag-handle').getBoundingClientRect().toJSON()})));
const h=agents[0].handle, target=agents[1].handle;
await page.mouse.move(h.left+h.width/2,h.top+h.height/2);await page.mouse.down();
await page.mouse.move(target.left+target.width/2,target.bottom+4,{steps:12});
await page.waitForSelector('#agents > .dragging');
assert(await page.evaluate(()=>{const row=document.querySelector('#agents > .dragging');renderAgents();return row===document.querySelector('#agents > .dragging')&&agentRenderPending;}),'agent redraw deferred');
await page.mouse.up();
await page.waitForFunction(id=>document.querySelector('#agents > .agent').dataset.id===id,agents[1].id);
assert.deepEqual((await audit()).posted.filter(p=>p.action==='/api/agents/arrange').at(-1).body.order,[agents[1].id,agents[0].id,...agents.slice(2).map(a=>a.id)]);
console.log('Existing agent arrangement PASS');
console.log('All account arrangement checks passed.');
await task.finish({keep:[]});
