from pathlib import Path
import subprocess,json,datetime,os,shutil,hashlib
r=Path('/home/winamp/.local/state/heliopause/m12-004');q=r/'qualification';driver=r/'run-qualified-consumers-timing.sh'
root=Path('/tmp/haa-m12-004-timing-native-e2e');root.mkdir(mode=0o700,exist_ok=False)
assert not root.is_symlink() and not list(root.iterdir())
s=driver.read_text();assert s.count('/tmp/haa-m12-004-native-e2e')==1
s=s.replace('/tmp/haa-m12-004-native-e2e',str(root));driver.write_text(s)
s=(r/'run-qualified-cli.sh').read_text().replace('qualified-permissions-','qualified-timing-').replace('"$original_root/baseline/cargo-source-policy-helper"','"$state_root/baseline/qualified-timing-policy-helper"').replace('qualification/ordinary-cli"','qualification/ordinary-cli-timing"')
(r/'run-qualified-cli-timing.sh').write_text(s)
working=3*3656681374+6556756381
free=os.statvfs('/mnt/c').f_bavail*os.statvfs('/mnt/c').f_frsize;assert free>=working+8*(1<<30)
(q/'timing-final-capacity.json').write_text(json.dumps({'physical_C_free':free,'case_working_set':working,'reserve':8*(1<<30),'fresh_full_root':str(root)},indent=2)+'\n')
steps=[
 ('cu126-timing-final-full','^TestLinuxPyTorchFullIntegration$','cu126','40m','bootstrap'),
 ('cpu-timing-final-full','^TestLinuxPyTorchFullIntegration$','cpu','15m','bootstrap'),
 ('terraform-timing-final-boundary','^TestLinuxTerraformProvider(Probe|Security)Integration$','none','5m','sandbox'),
 ('legacy-timing-final-direct','^TestLinux(GVisorLifecycle|GitHubReleaseELFDynamic|PyPISdistBuild|PyPIWheelDynamic|PythonRootPolicyTransactionSequence|PythonRenamedProcessTransaction|PythonPinnedLibutilTransaction|GoIsolatedResolver|GoIsolatedProjectResolver|GoSourceProjectSnapshot|GoDependencyFreeProjectSnapshot|CargoSourceProjectSnapshot)Integration$','none','15m','sandbox'),
 ('python-timing-final-proc-maps-cpu','^TestLinuxPythonProcMapsTransactionIntegration$','none','5m','sandbox'),
 ('python-timing-final-proc-maps-cu126','^TestLinuxPythonProcMapsTransactionIntegration$','cu126','5m','sandbox'),
 ('python-timing-final-proc-maps-cu130','^TestLinuxPythonProcMapsTransactionIntegration$','cu130','5m','sandbox'),
 ('python-timing-final-proc-maps-cu132','^TestLinuxPythonProcMapsTransactionIntegration$','cu132','5m','sandbox'),
 ('ordinary-timing-final-product-cli',None,None,None,None),
 ('go-cargo-timing-final-cli','^TestLinuxGo(GetDownload|TransitiveGetDownload|DependencyFreeDownload|DependencyFreeBuildCLI|PublicBuildCLI|TransitiveBuildCLI|InvalidTestdataBuildCLI|CgoBuildCLI|LibraryBuildCLI|MixedPackagesBuildCLI|MixedInvalidPackageBuildCLI|BuildSecurityCLI)Integration$|^TestLinuxCargo(Add|TransitiveAdd)Integration$','none','30m','bootstrap'),
 ('cargo-timing-final-build-cli','^TestLinuxCargo.*Build.*CLIIntegration$|^TestLinuxCargoProcMacroSecurityCLIIntegration$','none','15m','bootstrap'),
 ('promotion-timing-final-consumers','^TestLinux(NPMPromotion|PyPIPromotion)Integration$','none','5m','promotion'),
]
status=[]
for name,regex,profile,timeout,package in steps:
 subprocess.run(['python3',str(r/'verify-qualified-candidate.py')],check=True)
 cmd=['bash',str(r/'run-qualified-cli-timing.sh')] if regex is None else ['bash',str(driver),regex,profile,timeout,package]
 record={'candidate':'final-executable-candidate-timing.json','name':name,'command':cmd,'status':'RUNNING','started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat()};status.append(record)
 (q/'timing-final-campaign.json').write_text(json.dumps(status,indent=2)+'\n')
 log=q/(name+'-runtime.log')
 with log.open('xb') as f: result=subprocess.run(cmd,stdout=f,stderr=subprocess.STDOUT,cwd='/tmp/haa-m12-004-source')
 record.update(status='PASS' if result.returncode==0 else 'FAIL',exit_code=result.returncode,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),raw_log=log.name,raw_sha256=hashlib.sha256(log.read_bytes()).hexdigest())
 custody=q/(name+'-custody');custody.mkdir(mode=0o700)
 for pattern in ['*-after.modes','*.log','*.pid','*.wait','helper-launch*','helper-stop*']:
  for p in (r/'lifecycle').glob(pattern):
   if p.is_file() and p.stat().st_size<4*1024*1024:shutil.copyfile(p,custody/p.name)
 (q/'timing-final-campaign.json').write_text(json.dumps(status,indent=2)+'\n')
 print(name,record['status'],result.returncode,flush=True)
 if result.returncode:raise SystemExit(result.returncode)
