// Isolated API fixture for exercising the real UI with ego-browser (no browser
// dependency). Run: node internal/gui/tests/account-arrange-fixture.cjs
// Never reads user configuration or credentials. Writes are in memory only.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const assets = path.resolve(__dirname, '../assets');
const base = {icon:'generic', chat:'https://example.invalid/v1', models:[], agents:[], fallback:[], headers:{}, key:{set:true, masked:'fixture'}, routing:''};
const logins = Array.from({length:18}, (_, i) => ({agent:'antigravity', user:`account-${String(i+1).padStart(2,'0')}@example.test`, plan:'Pro', active:i===0, on:i!==2}));
const providers = {providers:[
 {...base, id:'antigravity', name:'Antigravity', keyList:[], account:{agent:'antigravity',agentName:'Antigravity',user:logins[0].user,logins}},
 {...base, id:'relay', name:'Relay', keyList:Array.from({length:5}, (_,i)=>({id:`key-${i+1}`,name:`Key ${i+1}`,masked:`test-…${i+1}`,active:i===0,on:i!==2}))}
],presets:[],excluded:[],gateway:{running:false,window:true}};
const state={agents:['alpha','beta','gamma'].map(id=>({id,name:id,icon:'generic',path:'/fixture',fields:[]})),profiles:[],settings:{lang:'zh',theme:'dark'}};
let posted = [], failNext = false;
const json = (res, data, code=200) => {res.writeHead(code, {'Content-Type':'application/json'});res.end(JSON.stringify(data));};
http.createServer(async(req,res)=>{
 try {
 const url = new URL(req.url,'http://localhost');
 if(url.pathname==='/boot.js') {res.setHeader('Content-Type','text/javascript'); return res.end('window.bootPrefs={lang:"zh",theme:"dark",web:true};');}
 if(url.pathname==='/wails/runtime.js') {res.setHeader('Content-Type','text/javascript');return res.end('export const Window={};');}
 if(url.pathname==='/api/state') return json(res,state);
 if(url.pathname==='/api/providers') return json(res,providers);
 if(url.pathname==='/api/groups') return json(res,{groups:[]});
 if(url.pathname==='/api/login/usage') return json(res,Object.fromEntries(logins.map((l,i)=>[l.user,{windows:Array.from({length:i%3+1},()=>({label:'Weekly',used:10,limit:100}))}])));
 if(url.pathname==='/__qa') return json(res,{posted,providers});
 if(url.pathname==='/__fail') {failNext=true; return json(res,{});}
 if(req.method==='POST' && url.pathname.startsWith('/api/')) {
 let raw=''; for await(const chunk of req) raw+=chunk;
 const body=JSON.parse(raw||'{}');posted.push({action:url.pathname,body});
 if(url.pathname==='/api/agents/arrange') {state.settings.agentOrder=body.order;return json(res,state.settings);}
 const p=providers.providers.find(p=>p.id===body.id);
 if(url.pathname==='/api/provider/arrange') {
  if(failNext) {failNext=false; return json(res,{error:'Fixture save failure'},500);}
  p.accountOrder=body.accountOrder;return json(res,{accountOrder:p.accountOrder});
 }
 if(url.pathname==='/api/provider/route') p.routing=body.routing;
 return json(res,providers);
 }
 if(url.pathname.startsWith('/api/')) return json(res,{});
 const file=path.resolve(assets, '.'+(url.pathname==='/'?'/index.html':url.pathname));
 if(!file.startsWith(assets+path.sep)) {res.writeHead(403);return res.end();}
 res.setHeader('Content-Type',({'.js':'text/javascript','.css':'text/css','.html':'text/html','.svg':'image/svg+xml','.png':'image/png'})[path.extname(file)]||'application/octet-stream');
 fs.createReadStream(file).on('error',()=>{res.statusCode=404;res.end();}).pipe(res);
 } catch(e) {json(res,{error:e.message},500);}
}).listen(18427,'127.0.0.1',()=>console.log('Isolated Magpie UI: http://127.0.0.1:18427/?view=providers'));
