from pathlib import Path
import json,hashlib,shutil,re,subprocess,stat
r=Path('/home/winamp/.local/state/heliopause/m12-004');q=r/'qualification';b=r/'baseline';d=q/'portable-draft';raw=d/'raw'
v=json.loads((q/'timing-final-campaign.json').read_text());assert len(v)==12 and all(x['status']=='PASS' and x['exit_code']==0 for x in v)
subprocess.run(['python3',str(r/'verify-qualified-candidate.py')],check=True)
subprocess.run(['python3',str(r/'summarize-qualification.py')],check=True)
runs=json.loads((q/'actual-scope-summary.json').read_text());assert all(x['terminal_restore_confirmed'] for x in runs if x['status']=='PASS')
for x in runs:shutil.copyfile(q/x['raw_log'],raw/x['raw_log'])
for name in ['campaign.json','continuation.json','timing-final-campaign.json','actual-scope-summary.json','pressure.jsonl','timing-final-pressure.jsonl','campaign-capacity.json','continuation-capacity.json','timing-final-capacity.json']:
 shutil.copyfile(q/name,d/name)
for name in ['final-executable-candidate-initial.json','final-executable-candidate-permissions.json','final-executable-candidate-timing.json']:
 p=q/name;c=json.loads(p.read_text());c['local_receipt']={'name':p.name,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()}
 c['binaries']=[{'role':Path(x['path']).name,'sha256':x['sha256'],'mode':oct(stat.S_IMODE(Path(x['path']).stat().st_mode))} for x in c['binaries']]
 (d/name).write_text(json.dumps(c,indent=2)+'\n')
for pattern in ['cpu-timing-*.log','cpu-read-diagnostic-*.log','cpu-install-diagnostic-*.log','provider-init-timing-*.log']:
 for p in b.glob(pattern):shutil.copyfile(p,raw/p.name)
for name in ['cpu-read-diagnostic-result.json','cpu-install-diagnostic-first-result.json','cpu-install-diagnostic-result-v2.json','cpu-install-diagnostic-pressure-v2.jsonl']:
 shutil.copyfile(b/name,d/name)
for name in ['run-qualified-consumers-timing.sh','run-qualified-cli-timing.sh','run-provider-init-timing.sh','run-qualification-timing-final.py','verify-qualified-candidate.py','make-timing-portable-evidence.py']:
 shutil.copyfile(r/name,d/name)
provider={}
for label,name,test,cases in [('positive','provider-init-timing-runtime.log','TestLinuxTerraformInitIntegration',6),('negative','provider-init-timing-negative-runtime.log','TestLinuxTerraformInitNegativeIntegration',4)]:
 p=b/name;s=p.read_text();matches=re.findall(r'^--- PASS: '+test+r' \(([0-9.]+)s\)',s,re.M);assert len(matches)==1
 assert 'wait=0' in s and 'cleanup: original=0 helper_state=CONFIRMED_STOPPED helper_stop=0' in s and 'runtime before bytes/identity/modes verified' in s
 provider[label]={'raw':'raw/'+name,'raw_sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'seconds':float(matches[0]),'cases':cases,'original12_restored':True,'helper_wait':0}
 shutil.copyfile(p,raw/name)
full={}
for label,name,limit in [('cpu','cpu-timing-final-full',900),('cu126','cu126-timing-final-full',2400)]:
 x=next(x for x in runs if x['name']==name);assert len(x['tests'])==1 and x['tests'][0]['seconds']<limit and x['terminal_restore_confirmed']
 full[label]={'status':'PASS','test_seconds':x['tests'][0]['seconds'],'timeout_seconds':limit,'tool_wrapper_exit':x['exit_code'],'raw':'raw/'+x['raw_log'],'raw_sha256':x['raw_sha256'],'terminal_restore_confirmed':True,'candidate':'final-executable-candidate-timing.json'}
 s=(q/x['raw_log']).read_text();roots=re.findall(r'retained integration evidence: (\S+)',s);assert len(roots)==1
 case=Path(roots[0]);evidence=case/'cache/heliopause/evidence';records=[];known={}
 out=d/(label+'-stored-evidence');out.mkdir(exist_ok=False)
 for p in sorted(evidence.glob('*/*.json')):
  record=json.loads(p.read_text());relative=p.relative_to(evidence);target=out/relative;target.parent.mkdir(exist_ok=True);shutil.copyfile(p,target)
  ref={'path':target.relative_to(d).as_posix(),'sha256':hashlib.sha256(p.read_bytes()).hexdigest()};records.append(ref)
  if 'declared-sha256' in record['kind']:known.setdefault(record['sha256'],[]).append((record,ref))
 inputs=[]
 for directory in sorted((case/'cache/heliopause/intake').iterdir()):
  p=directory/'wheel.whl'
  if not p.exists():continue
  s=p.lstat();assert stat.S_ISREG(s.st_mode) and s.st_nlink==1
  with p.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
  filename=(directory/'filename').read_text();selected=known.get(digest,[])
  if selected:
   identities={(z[0]['source_id'],z[0]['name'],z[0]['version']) for z in selected};assert len(identities)==1
   source,name,version=next(iter(identities));role='promotion-set';refs=[z[1] for z in selected]
  else:
   assert label=='cu126' and filename=='numpy-2.4.6-cp314-cp314-manylinux_2_27_x86_64.manylinux_2_28_x86_64.whl' and digest=='a2c306dea656c12c68f51f4cea133cbe78ca7435eb28c735eac1d3ebe73be6e8'
   source,name,version='pypi','numpy','2.4.6';role='inspection-prerequisite-excluded-from-promotion';refs=[]
  inputs.append({'role':role,'source_id':source,'name':name,'version':version,'filename':filename,'sha256':digest,'bytes':s.st_size,'declared_sha256_evidence':refs})
 target_records=[];target=case/'target'
 for p in sorted(target.rglob('*')):
  if p.is_dir():continue
  s=p.lstat();assert stat.S_ISREG(s.st_mode) and s.st_nlink==1
  with p.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
  target_records.append({'path':p.relative_to(target).as_posix(),'sha256':digest,'bytes':s.st_size,'mode':oct(stat.S_IMODE(s.st_mode))})
 (d/(label+'-input-target-inventory.json')).write_text(json.dumps({'scope':'Observed authenticated intake is linked to actual stored declared checksum evidence; computed observed hashes alone are not source authorization','inputs':inputs,'stored_evidence':records,'target':target_records},indent=2)+'\n')
 full[label]['input_target_inventory']=label+'-input-target-inventory.json'
(d/'full-results.json').write_text(json.dumps(full,indent=2)+'\n')
original=r.parent/'m12-003/source-lifecycle/runtime-before.json'
shutil.copyfile(original,d/'original-runtime-before.json')
shutil.copyfile(r.parent/'m12-003/kernel-runtime-candidate/verified-bundle/gvisor-bundle.manifest.json',d/'qualified-runtime-bundle.manifest.json')
for name in ['installed-before.sha256','installed-before.modes','pod-init-before.sha256','pod-init-before.modes','runtime-restore-list.txt']:
 shutil.copyfile(r/'lifecycle'/name,d/name)
result={'work_item':'M12-004','status':'QUALIFICATION_PASSED_PENDING_CLOSURE_CHECKS','IMPLEMENTED':'YES','WIRED':'YES','QUALIFIED':'YES (Linux/amd64 local)','ACCEPTANCE_CLOSED':'NO','baseline_head':'8c56f873eb7ce395af4ba984190413492b096e63','source_checkpoint':'e788c1a','scope':'Public Provider installation only; official random3.7.2 and partner integrations/github6.6.0 actual signed/static/isolated fixed-help/Policy/cache/guarded initial+retained complete project installation. RPC/plan/apply/cloud behavior NOT_ATTESTED.','candidate':'final-executable-candidate-timing.json','exact_source_count':len(json.loads((q/'final-executable-candidate-timing.json').read_text())['source']),'provider_init':provider,'actual_scope':'actual-scope-summary.json','full_results':full,'causal_proofs':'causal-proofs.json','prior_cu126_failure':{'raw':'raw/cu126-full-runtime.log','status':'FAIL','sample_gap_ns':1183829770,'root_cause':'UNCONFIRMED; subsequent PASS and timing diagnostics are not a claimed fix'},'prior_deliberate_hold':{'raw':'raw/python-proc-maps-cu132-runtime.log','exit':98,'scope':'pre-install preparation hold, NOT_REACHED; after-hold actual PASS recorded separately'},'native_macos':'NOT_RUN; Darwin TEST compile only','remote_required':'NOT_RUN','cu130_cu132':'Focused common-path only; full NOT_RUN','release_support':'Broader release matrix remains M12-005','missing':['Complete candidate documentation/workflow/security/staged checks and local Korean commit'],'delivery':'NO PUSH / NO REMOTE CI / NO MERGE / NO AGENTS; M12-005 NOT_STARTED'}
(d/'result.json').write_text(json.dumps(result,indent=2)+'\n')
entries=[]
for p in sorted(d.rglob('*')):
 if p.is_file() and p.name!='raw-inventory.json':entries.append({'path':p.relative_to(d).as_posix(),'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'bytes':p.stat().st_size})
(d/'raw-inventory.json').write_text(json.dumps(entries,indent=2)+'\n')
print('Portable qualification evidence ready:',len(entries),'files; acceptance closure remains pending')
